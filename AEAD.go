package spongebob

import (
	"crypto/cipher"
	"crypto/subtle"
	"errors"

	streambob "github.com/MHSarmadi/SpongeBob/StreamBob"
	"github.com/MHSarmadi/SpongeBob/internal"
)

const (
	NonceSize = 12
	TagSize   = 16
)

var ErrOpen = errors.New("spongebob: message authentication failed")

type AEAD struct {
	// state is the template and must not be modified once initialized.
	state streambob.Keyed
}

var _ cipher.AEAD = (*AEAD)(nil)

func NewAEAD(key []byte, ctx string) (*AEAD, error) {
	a := &AEAD{}
	if err := a.state.CustomInit(streambob.Variant256, key, ctx, internal.FULLCPHR); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *AEAD) NonceSize() int { return NonceSize }
func (a *AEAD) Overhead() int  { return TagSize }

func (a *AEAD) Seal(dst, nonce, plaintext, additionalData []byte) []byte {
	// copy the state, so multi operations with the same key+ctx is possible.
	// a.state is the template and must not be modified.
	state := a.state
	defer state.Wipe()

	if len(nonce) != NonceSize {
		panic("spongebob: bad nonce length")
	}

	half := (len(plaintext) + 1) / 2
	ret, out := sliceForAppend(dst, len(plaintext)+TagSize)
	copy(out, plaintext)
	left, right := out[:half], out[half:len(plaintext)]
	tag := out[len(plaintext):]
	for i := range tag {
		tag[i] = 0
	}

	state.AbsorbCustomDomain(additionalData, internal.ADD_DATA)
	state.Permute()
	state.AbsorbCustomDomain(nonce, internal.N0NC3___)
	state.Permute()

	bak := state
	defer bak.Wipe()

	state = bak
	state.Absorb(left)
	state.Squeeze(right)

	state = bak
	state.Absorb(right)
	state.Squeeze(left)

	state = bak
	state.Absorb(left)
	state.Squeeze(right)

	state = bak
	state.AbsorbCustomDomain(left, internal.AUTH_TAG)
	state.AbsorbCustomDomain(right, internal.AUTH_TAG)
	state.Squeeze(tag)

	return ret
}

func (a *AEAD) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	state := a.state // copy the state, so multi operations with the same key+ctx is possible.
	defer state.Wipe()

	if len(nonce) != NonceSize {
		panic("spongebob: bad nonce length")
	}

	plaintext_len := len(ciphertext) - TagSize
	if plaintext_len < 0 {
		return nil, ErrOpen
	}

	half := (plaintext_len + 1) / 2
	ret, out := sliceForAppend(dst, plaintext_len)
	copy(out, ciphertext)
	left, right := out[:half], out[half:]
	tag := ciphertext[plaintext_len:]

	state.AbsorbCustomDomain(additionalData, internal.ADD_DATA)
	state.Permute()
	state.AbsorbCustomDomain(nonce, internal.N0NC3___)
	state.Permute()

	bak := state
	defer bak.Wipe()

	var expected_tag [TagSize]byte
	state = bak
	state.AbsorbCustomDomain(left, internal.AUTH_TAG)
	state.AbsorbCustomDomain(right, internal.AUTH_TAG)
	state.Squeeze(expected_tag[:])

	if subtle.ConstantTimeCompare(tag, expected_tag[:]) == 0 {
		return nil, ErrOpen
	}

	state = bak
	state.Absorb(left)
	state.Squeeze(right)

	state = bak
	state.Absorb(right)
	state.Squeeze(left)

	state = bak
	state.Absorb(left)
	state.Squeeze(right)

	return ret, nil
}

func (a *AEAD) Wipe() {
	a.state.Wipe()
}

// sliceForAppend takes a slice and a requested number of bytes. It returns a
// slice with the contents of the given slice followed by that many bytes and a
// second slice that aliases into it and contains only the extra bytes. If the
// original slice has sufficient capacity then no allocation is performed.
func sliceForAppend(in []byte, n int) (head, tail []byte) {
	if total := len(in) + n; cap(in) >= total {
		head = in[:total]
	} else {
		head = make([]byte, total)
		copy(head, in)
	}
	tail = head[len(in):]
	return
}
