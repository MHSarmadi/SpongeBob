package streambob

import (
	"github.com/MHSarmadi/SpongeBob/internal"
)

// Keyed is a streaming sponge MAC, i.e. a keyed hash.
//
// The zero value is not ready to use. Construct with NewKeyed128,
// NewKeyed256, or NewKeyed480, or call Init128/Init256/Init480 on a zero
// value. Input is fed via Absorb (Write) and output produced via Squeeze
// (Read), with the same streaming semantics as Unkeyed.
//
// The secret is absorbed at the maximum capacity rate (960 bits), isolating
// it from message-length effects; key length is not limited and any length
// up to and beyond the rate is well-defined. Keyed output for the same
// secret, context, and message is indistinguishable from random for parties
// without the secret.
//
// Keyed is not safe for concurrent use.
type Keyed struct {
	state      internal.State
	variant    variant
	rate_bytes int
	squeezing  bool
	absorbing  bool
}

// NewKeyed128 returns a Keyed MAC with 256-bit capacity (128-bit collision,
// 256-bit preimage security) bound to secret and context.
func NewKeyed128(secret []byte, context string) *Keyed {
	h := &Keyed{}
	h.Init128(secret, context)
	return h
}

// NewKeyed256 returns a Keyed MAC with 512-bit capacity (256-bit collision,
// 512-bit preimage security) bound to secret and context.
func NewKeyed256(secret []byte, context string) *Keyed {
	h := &Keyed{}
	h.Init256(secret, context)
	return h
}

// NewKeyed480 returns a Keyed MAC with 960-bit capacity (480-bit collision,
// 960-bit preimage security) bound to secret and context. Roughly 12×
// slower than Variant128; use only when the higher security bound is
// required.
func NewKeyed480(secret []byte, context string) *Keyed {
	h := &Keyed{}
	h.Init480(secret, context)
	return h
}

// Init128 initializes h as a Variant128 (256-bit capacity) keyed hasher
// bound to secret and context. Panics if h is already initialized; call
// Wipe first.
func (h *Keyed) Init128(secret []byte, context string) {
	h.init(Variant128, secret, context, internal.KEYED___)
}

// Init256 initializes h as a Variant256 (512-bit capacity) keyed hasher
// bound to secret and context. Panics if h is already initialized; call
// Wipe first.
func (h *Keyed) Init256(secret []byte, context string) {
	h.init(Variant256, secret, context, internal.KEYED___)
}

// Init480 initializes h as a Variant480 (960-bit capacity) keyed hasher
// bound to secret and context. Panics if h is already initialized; call
// Wipe first.
func (h *Keyed) Init480(secret []byte, context string) {
	h.init(Variant480, secret, context, internal.KEYED___)
}

// CustomInit is Init128/256/480 but with a caller-chosen domain separation
// constant. The domain is XORed into the capacity on every permutation,
// isolating this MAC from any other primitive built on the same sponge.
// Returns errInvalidVariant if v is not one of the defined variants.
//
// Use the fixed Init methods unless building a new primitive on top of
// Keyed; the domain constants are part of the wire format, and changing
// them silently changes every tag.
func (h *Keyed) CustomInit(v variant, secret []byte, context string, DOMAIN uint64) error {
	switch v {
	case Variant128, Variant256, Variant480:
		h.init(v, secret, context, DOMAIN)
		return nil
	default:
		return errInvalidVariant
	}
}

func (h *Keyed) init(v variant, secret []byte, context string, DOMAIN uint64) {
	h.state.Init([4]uint64{internal.SP0NGB0B, internal.PR1M1T1V, internal.SARMADI_, DOMAIN})

	h.state.BeginAbsorb(internal.RATE_OF_CAP256_BYTES, false, internal.C0NT3XT_)
	h.state.Absorb([]byte(context))
	h.state.BeginAbsorb(internal.RATE_OF_CAP960_BYTES, false, internal.S3CR3T__)
	h.state.Absorb(secret)

	h.variant = v
	switch v {
	case Variant128:
		h.rate_bytes = internal.RATE_OF_CAP256_BYTES
	case Variant256:
		h.rate_bytes = internal.RATE_OF_CAP512_BYTES
	case Variant480:
		h.rate_bytes = internal.RATE_OF_CAP960_BYTES
	default:
		panic("streamBob: unknown variant")
	}
}

// Absorb feeds p into the sponge. Calls accumulate: multiple Absorb calls
// absorb as one continuous stream. Empty p is a no-op (the state is not
// advanced).
//
// If h is currently squeezing, Absorb discards any pending squeeze bytes and
// starts a new absorb session on the current sponge state — use Wipe + Init
// for a fully fresh MAC instead.
//
// Returns errUninitialized if h has not been initialized or has been Wiped.
func (h *Keyed) Absorb(p []byte) (int, error) {
	if h.variant == 0 {
		return 0, errUninitialized
	}
	if h.squeezing {
		h.state.Reset()
	}
	if !h.absorbing {
		h.state.BeginAbsorb(h.rate_bytes, true, internal.PLA1NTXT)
		h.absorbing = true
		h.squeezing = false
	}
	if len(p) == 0 {
		return 0, nil // no-op: don't unseal for an empty absorb
	}
	h.state.Absorb(p)
	return len(p), nil
}

// Write implements io.Writer and is an alias for Absorb.
func (h *Keyed) Write(p []byte) (int, error) {
	return h.Absorb(p)
}

// Permute advances the sponge by exactly one permutation. If a streamed
// absorb is in progress, Permute first seals it (padding and absorbing the
// tail); otherwise it permutes the current state directly. Useful for
// field-separated protocols, e.g. Absorb(a); Permute(); Absorb(b).
//
// During squeezing, Permute advances past the current output block and
// discards any buffered output bytes not yet read.
//
// Returns errUninitialized if h has not been initialized or has been Wiped.
func (h *Keyed) Permute() error {
	if h.variant == 0 {
		return errUninitialized
	}
	if h.absorbing {
		h.state.SealAbsorb()
		h.absorbing = false
	} else {
		h.state.Permute()
	}
	return nil
}

// Squeeze writes len(p) bytes of tag stream into p. The first call after
// absorb seals the pending absorb, then reads from the sponge; subsequent
// calls continue the output stream. Output is chunk-invariant.
//
// p is XORed into: pass a zeroed slice for pure extraction, or pass
// plaintext to encipher in place.
//
// Returns errUninitialized if h has not been initialized or has been Wiped.
func (h *Keyed) Squeeze(p []byte) (int, error) {
	if h.variant == 0 {
		return 0, errUninitialized
	}
	if h.absorbing {
		h.state.SealAbsorb()
		h.absorbing = false
	}
	if !h.squeezing {
		h.state.BeginSqueeze(h.rate_bytes, true)
		h.squeezing = true
	}
	h.state.Squeeze(p)
	return len(p), nil
}

// Read implements io.Reader and is an alias for Squeeze.
func (h *Keyed) Read(p []byte) (int, error) {
	return h.Squeeze(p)
}

// Wipe zeroes the sponge state, internal buffers, bookkeeping, and the
// reference to the secret held in h's absorbed state. Call this once the
// MAC is no longer needed. After Wipe, h must be re-initialized with
// Init{128,256,480} or CustomInit.
func (h *Keyed) Wipe() {
	h.state.Wipe()
	h.variant = 0
	h.squeezing = false
	h.absorbing = false
}
