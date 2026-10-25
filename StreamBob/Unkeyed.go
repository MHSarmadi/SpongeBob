package streambob

import (
	"github.com/MHSarmadi/SpongeBob/internal"
)

type Unkeyed struct {
	state      internal.State
	variant    variant
	rate_bytes int
}

func NewUnkeyed128(context string) *Unkeyed {
	h := &Unkeyed{}
	h.Init128(context)
	return h
}
func NewUnkeyed256(context string) *Unkeyed {
	h := &Unkeyed{}
	h.Init256(context)
	return h
}
func NewUnkeyed480(context string) *Unkeyed {
	h := &Unkeyed{}
	h.Init480(context)
	return h
}

func (h *Unkeyed) Init128(context string) {
	h.init(Variant128, context, internal.UNKEYED_)
}
func (h *Unkeyed) Init256(context string) {
	h.init(Variant256, context, internal.UNKEYED_)
}
func (h *Unkeyed) Init480(context string) {
	h.init(Variant480, context, internal.UNKEYED_)
}

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
	h.state.Init([4]uint64{internal.SP0NGB0B, internal.PR1M1T1V, internal.SARMADI_, DOMAIN}, true)

	h.state.Absorb256([]byte(context), internal.C0NT3XT_)
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

func (h *Unkeyed) Absorb(p []byte) (int, error) {
	if h.variant == 0 {
		return 0, errUninitialized
	}
	if len(p) == 0 {
		return 0, nil // no-op: don't unseal for an empty absorb
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

func (h *Unkeyed) Write(p []byte) (int, error) {
	return h.Absorb(p)
}

func (h *Unkeyed) Permute() error {
	if h.variant == 0 {
		return errUninitialized
	}
	if !h.state.SealAbsorb() {
		h.state.Permute()
	}
	return nil
}

func (h *Unkeyed) Squeeze(p []byte) (int, error) {
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

func (h *Unkeyed) Read(p []byte) (int, error) {
	return h.Squeeze(p)
}

func (h *Unkeyed) Wipe() {
	h.state.Wipe()
	h.variant = 0
}
