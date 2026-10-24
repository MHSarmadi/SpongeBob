package test

import (
	"bytes"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
)

type rateCase struct {
	name     string
	rateSize int
	fn       func(*internal.State, []byte)
}

func squeezeCases() []rateCase {
	return []rateCase{
		{"256", internal.RATE_OF_CAP256_BYTES, (*internal.State).Squeeze256},
		{"512", internal.RATE_OF_CAP512_BYTES, (*internal.State).Squeeze512},
		{"960", internal.RATE_OF_CAP960_BYTES, (*internal.State).Squeeze960},
	}
}

func TestSqueeze_EmptyDoesNotPermute(t *testing.T) {
	for _, cache := range []bool{false, true} {
		name := map[bool]string{false: "uncached", true: "cached"}[cache]
		t.Run(name, func(t *testing.T) {
			var initials = [4]uint64{0xDEADBEEFCAFEBABE, 0x1122334455667788, 0x9988776655443322}
			s := internal.State{}

			s.Init(initials, cache)

			var before [4]byte
			clone_b := s.Clone()
			clone_b.Squeeze256(before[:])

			s.Squeeze256(nil)
			s.Squeeze256([]byte{})
			s.Squeeze512(nil)
			s.Squeeze512([]byte{})
			s.Squeeze960(nil)
			s.Squeeze960([]byte{})

			var after [4]byte
			clone_a := s.Clone()
			clone_a.Squeeze256(after[:])

			if after != before {
				t.Error("empty squeeze permuted state")
			}
		})
	}
}

func TestSqueeze_XORSemantics(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			s1 := internal.State{}
			s1.Init(zero_initials, false)
			raw := make([]byte, r.rateSize)
			r.fn(&s1, raw)

			s2 := internal.State{}
			s2.Init(zero_initials, false)
			ff := bytes.Repeat([]byte{0xFF}, r.rateSize)
			r.fn(&s2, ff)

			for i := range raw {
				want := raw[i] ^ 0xFF
				if ff[i] != want {
					t.Fatalf("byte %d: got %#x want %#x", i, ff[i], want)
				}
			}
		})
	}
}

func TestSqueeze_Cached_ChunkIndependence(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			totals := []int{1, 7, r.rateSize - 1, r.rateSize, r.rateSize + 1, 2 * r.rateSize, 3*r.rateSize + 5}
			for _, total := range totals {
				if total <= 0 {
					continue
				}
				s1 := internal.State{}
				s1.Init(zero_initials, true)
				one := make([]byte, total)
				r.fn(&s1, one)

				for _, split := range []int{1, 7, total / 2, total - 1, r.rateSize, total - r.rateSize} {
					if split <= 0 || split >= total {
						continue
					}
					s2 := internal.State{}
					s2.Init(zero_initials, true)
					splitOut := make([]byte, total)
					r.fn(&s2, splitOut[:split])
					r.fn(&s2, splitOut[split:])
					if !bytes.Equal(one, splitOut) {
						t.Errorf("total=%d split=%d: cached chunk mismatch", total, split)
					}
				}
			}
		})
	}
}

func TestSqueeze_Uncached_RateAligned(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			s1 := internal.State{}
			s1.Init(zero_initials, false)
			one := make([]byte, 2*r.rateSize)
			r.fn(&s1, one)

			s2 := internal.State{}
			s2.Init(zero_initials, false)
			two := make([]byte, 2*r.rateSize)
			r.fn(&s2, two[:r.rateSize])
			r.fn(&s2, two[r.rateSize:])
			if !bytes.Equal(one, two) {
				t.Error("rate-aligned split mismatch")
			}
		})
	}
}

func TestSqueeze_Uncached_UnalignedDiffers(t *testing.T) {
	// Sanity: unaligned splits DO differ (documents the contract).
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			if r.rateSize < 4 {
				t.Skip("rate too small for a meaningful unaligned split")
			}
			half := r.rateSize / 2
			s1 := internal.State{}
			s1.Init(zero_initials, false)
			one := make([]byte, r.rateSize)
			r.fn(&s1, one)

			s2 := internal.State{}
			s2.Init(zero_initials, false)
			two := make([]byte, r.rateSize)
			r.fn(&s2, two[:half])
			r.fn(&s2, two[half:])

			if bytes.Equal(one, two) {
				t.Error("unaligned split unexpectedly matches one-shot")
			}
		})
	}
}

func TestSqueeze_CachedVsUncached_RateAligned(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			sc := internal.State{}
			sc.Init(zero_initials, true)
			su := internal.State{}
			su.Init(zero_initials, false)
			outC := make([]byte, r.rateSize)
			outU := make([]byte, r.rateSize)
			r.fn(&sc, outC)
			r.fn(&su, outU)
			if !bytes.Equal(outC, outU) {
				t.Error("rate-aligned cached vs uncached mismatch")
			}
		})
	}
}
