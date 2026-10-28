package streambob

import (
	"github.com/MHSarmadi/SpongeBob/internal"
)

// Unkeyed is a streaming sponge hasher for unkeyed hashing (or XOF via Read).
//
// The zero value is not ready to use. Construct with NewUnkeyed128,
// NewUnkeyed256, or NewUnkeyed480, or call Init128/Init256/Init480 on a
// zero value. Input is fed via Absorb (Write) and output produced via
// Squeeze (Read). Absorb and Squeeze may be interleaved; switching from
// squeeze back to absorb discards any pending squeeze bytes and starts a
// new absorb session on the current sponge state.
//
// The context string is bound into the initial state and must match between
// parties that need to agree on output. It is not secret; use Keyed when a
// key is required.
//
// Unkeyed is not safe for concurrent use.
type Unkeyed struct {
	state      internal.State
	variant    variant
	rate_bytes int
	squeezing  bool
	absorbing  bool
}

// NewUnkeyed128 returns an Unkeyed with 256-bit capacity (128-bit collision,
// 256-bit preimage security) bound to context.
func NewUnkeyed128(context string) *Unkeyed {
	h := &Unkeyed{}
	h.Init128(context)
	return h
}

// NewUnkeyed256 returns an Unkeyed with 512-bit capacity (256-bit collision,
// 512-bit preimage security) bound to context.
func NewUnkeyed256(context string) *Unkeyed {
	h := &Unkeyed{}
	h.Init256(context)
	return h
}

// NewUnkeyed480 returns an Unkeyed with 960-bit capacity (480-bit collision,
// 960-bit preimage security) bound to context. Roughly 12× slower than
// Variant128 due to the narrow rate; use only when the higher security
// bound is required.
func NewUnkeyed480(context string) *Unkeyed {
	h := &Unkeyed{}
	h.Init480(context)
	return h
}

// Init128 initializes h as a Variant128 (256-bit capacity) unkeyed hasher
// bound to context. Panics if h is already initialized; call Wipe first.
func (h *Unkeyed) Init128(context string) {
	h.init(Variant128, context, internal.UNKEYED_)
}

// Init256 initializes h as a Variant256 (512-bit capacity) unkeyed hasher
// bound to context. Panics if h is already initialized; call Wipe first.
func (h *Unkeyed) Init256(context string) {
	h.init(Variant256, context, internal.UNKEYED_)
}

// Init480 initializes h as a Variant480 (960-bit capacity) unkeyed hasher
// bound to context. Panics if h is already initialized; call Wipe first.
func (h *Unkeyed) Init480(context string) {
	h.init(Variant480, context, internal.UNKEYED_)
}

// CustomInit is Init128/256/480 but with a caller-chosen domain separation
// constant. The domain is XORed into the capacity on every permutation,
// isolating this hasher from any other primitive built on the same sponge.
// Returns errInvalidVariant if v is not one of the defined variants.
//
// Use the fixed Init methods unless building a new primitive on top of
// Unkeyed; the domain constants are part of the wire format, and changing
// them silently changes every output.
func (h *Unkeyed) CustomInit(v variant, context string, DOMAIN uint64) error {
	switch v {
	case Variant128, Variant256, Variant480:
		h.init(v, context, DOMAIN)
		return nil
	default:
		return errInvalidVariant
	}
}

func (h *Unkeyed) init(v variant, context string, DOMAIN uint64) {
	h.state.Init([4]uint64{internal.SP0NGB0B, internal.PR1M1T1V, internal.SARMADI_, DOMAIN})

	h.state.BeginAbsorb(internal.RATE_OF_CAP256_BYTES, false, internal.C0NT3XT_)
	h.state.Absorb([]byte(context))

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
// for a fully fresh state instead.
//
// Returns errUninitialized if h has not been initialized or has been Wiped.
func (h *Unkeyed) Absorb(p []byte) (int, error) {
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
func (h *Unkeyed) Write(p []byte) (int, error) {
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
func (h *Unkeyed) Permute() error {
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

// Squeeze writes len(p) bytes of output into p. The first call after absorb
// seals the pending absorb, then reads from the sponge; subsequent calls
// continue the output stream. Output is chunk-invariant: splitting a Read
// into multiple calls produces the same concatenated bytes as a single call
// (unlike hash.Hash, this is true even for chunk sizes smaller than the
// rate).
//
// p is XORed into: pass a zeroed slice for pure extraction, or pass
// plaintext to encipher in place (stream-cipher mode).
//
// Returns errUninitialized if h has not been initialized or has been Wiped.
func (h *Unkeyed) Squeeze(p []byte) (int, error) {
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
func (h *Unkeyed) Read(p []byte) (int, error) {
	return h.Squeeze(p)
}

// Wipe zeroes the sponge state, internal buffers, and bookkeeping, and
// returns h to an unusable (zero) state. Call this once the hasher is no
// longer needed, especially when it has absorbed sensitive material.
//
// After Wipe, h must be re-initialized with Init{128,256,480} or CustomInit.
func (h *Unkeyed) Wipe() {
	h.state.Wipe()
	h.variant = 0
	h.squeezing = false
	h.absorbing = false
}
