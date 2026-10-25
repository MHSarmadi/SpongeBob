package streambob

import (
	"github.com/MHSarmadi/SpongeBob/internal"
)

type Keyed struct {
	state      internal.State
	variant    variant
	rate_bytes int
}

func NewKeyed128(secret []byte, context string) *Keyed {
	h := &Keyed{}
	h.Init128(secret, context)
	return h
}
func NewKeyed256(secret []byte, context string) *Keyed {
	h := &Keyed{}
	h.Init256(secret, context)
	return h
}
func NewKeyed480(secret []byte, context string) *Keyed {
	h := &Keyed{}
	h.Init480(secret, context)
	return h
}

func (h *Keyed) Init128(secret []byte, context string) {
	h.init(Variant128, secret, context, internal.KEYED___)
}
func (h *Keyed) Init256(secret []byte, context string) {
	h.init(Variant256, secret, context, internal.KEYED___)
}
func (h *Keyed) Init480(secret []byte, context string) {
	h.init(Variant480, secret, context, internal.KEYED___)
}

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
	h.state.Init([4]uint64{internal.SP0NGB0B, internal.PR1M1T1V, internal.SARMADI_, DOMAIN}, true)

	h.state.Absorb256([]byte(context), internal.C0NT3XT_)
	h.state.SealAbsorb()
	h.state.Absorb960(secret, internal.S3CR3T__)
	h.state.SealAbsorb()

	h.variant = v
	switch v {
	case Variant128:
		h.rate_bytes = internal.RATE_OF_CAP256_BYTES
	case Variant256:
		h.rate_bytes = internal.RATE_OF_CAP512_BYTES
	case Variant480:
		h.rate_bytes = internal.RATE_OF_CAP960_BYTES
	}
}

func (h *Keyed) Absorb(p []byte) (int, error) {
	if h.variant == 0 {
		return 0, errUninitialized
	}
	switch h.variant {
	case Variant128:
		h.state.Absorb256(p, internal.PLA1NTXT)
	case Variant256:
		h.state.Absorb512(p, internal.PLA1NTXT)
	case Variant480:
		h.state.Absorb960(p, internal.PLA1NTXT)
	}
	return len(p), nil
}

func (h *Keyed) Write(p []byte) (int, error) {
	return h.Absorb(p)
}

func (h *Keyed) Permute() error {
	if h.variant == 0 {
		return errUninitialized
	}
	if !h.state.SealAbsorb() {
		h.state.Permute()
	}
	return nil
}

func (h *Keyed) Squeeze(p []byte) (int, error) {
	if h.variant == 0 {
		return 0, errUninitialized
	}
	switch h.variant {
	case Variant128:
		h.state.Squeeze256(p)
	case Variant256:
		h.state.Squeeze512(p)
	case Variant480:
		h.state.Squeeze960(p)
	}
	return len(p), nil
}

func (h *Keyed) Read(p []byte) (int, error) {
	return h.Squeeze(p)
}

func (h *Keyed) Wipe() {
	h.state.Wipe()
	h.variant = 0
}
