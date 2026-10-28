package test

import (
	"bytes"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
)

type rateCase struct {
	name     string
	rateSize int
}

func squeezeCases() []rateCase {
	return []rateCase{
		{"256", internal.RATE_OF_CAP256_BYTES},
		{"512", internal.RATE_OF_CAP512_BYTES},
		{"960", internal.RATE_OF_CAP960_BYTES},
	}
}

func TestSqueeze_EmptyDoesNotPermute(t *testing.T) {
	for _, r := range squeezeCases() {
		for _, stream := range []bool{false, true} {
			t.Run(r.name+"/"+modeName(stream), func(t *testing.T) {
				var s1, s2 internal.State
				s1.Init(zero_initials)
				s2.Init(zero_initials)
				s1.BeginSqueeze(r.rateSize, stream)
				s2.BeginSqueeze(r.rateSize, stream)

				s1.Squeeze(nil)
				s1.Squeeze([]byte{})
				s1.Squeeze(nil)

				out1 := make([]byte, 32)
				out2 := make([]byte, 32)
				s1.Squeeze(out1)
				s2.Squeeze(out2)

				if !bytes.Equal(out1, out2) {
					t.Fatal("empty squeeze advanced the state")
				}
			})
		}
	}
}

func TestSqueeze_XORSemantics(t *testing.T) {
	for _, r := range squeezeCases() {
		for _, stream := range []bool{false, true} {
			t.Run(r.name+"/"+modeName(stream), func(t *testing.T) {
				var s1, s2 internal.State
				s1.Init(zero_initials)
				s2.Init(zero_initials)
				s1.BeginSqueeze(r.rateSize, stream)
				s2.BeginSqueeze(r.rateSize, stream)

				raw := make([]byte, r.rateSize)
				ff := bytes.Repeat([]byte{0xFF}, r.rateSize)
				s1.Squeeze(raw)
				s2.Squeeze(ff)

				for i := range raw {
					want := raw[i] ^ 0xFF
					if ff[i] != want {
						t.Fatalf("byte %d: got %#x want %#x", i, ff[i], want)
					}
				}
			})
		}
	}
}

func TestSqueeze_Streaming_ChunkIndependence(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			totals := []int{1, 7, r.rateSize - 1, r.rateSize, r.rateSize + 1, 2 * r.rateSize, 3*r.rateSize + 5}
			for _, total := range totals {
				if total <= 0 {
					continue
				}
				var s1 internal.State
				s1.Init(zero_initials)
				s1.BeginSqueeze(r.rateSize, true)
				one := make([]byte, total)
				s1.Squeeze(one)

				for _, split := range []int{1, 7, total / 2, total - 1, r.rateSize, total - r.rateSize} {
					if split <= 0 || split >= total {
						continue
					}
					var s2 internal.State
					s2.Init(zero_initials)
					s2.BeginSqueeze(r.rateSize, true)
					splitOut := make([]byte, total)
					s2.Squeeze(splitOut[:split])
					s2.Squeeze(splitOut[split:])
					if !bytes.Equal(one, splitOut) {
						t.Errorf("total=%d split=%d: streaming chunk mismatch", total, split)
					}
				}
			}
		})
	}
}

func TestSqueeze_NonStreaming_RateAligned(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			var s1 internal.State
			s1.Init(zero_initials)
			s1.BeginSqueeze(r.rateSize, false)
			one := make([]byte, 2*r.rateSize)
			s1.Squeeze(one)

			var s2 internal.State
			s2.Init(zero_initials)
			s2.BeginSqueeze(r.rateSize, false)
			two := make([]byte, 2*r.rateSize)
			s2.Squeeze(two[:r.rateSize])
			s2.Squeeze(two[r.rateSize:])
			if !bytes.Equal(one, two) {
				t.Error("rate-aligned split mismatch")
			}
		})
	}
}

func TestSqueeze_NonStreaming_UnalignedDiffers(t *testing.T) {
	// Sanity: unaligned splits DO differ (documents the contract).
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			if r.rateSize < 4 {
				t.Skip("rate too small for a meaningful unaligned split")
			}
			half := r.rateSize / 2
			var s1 internal.State
			s1.Init(zero_initials)
			s1.BeginSqueeze(r.rateSize, false)
			one := make([]byte, r.rateSize)
			s1.Squeeze(one)

			var s2 internal.State
			s2.Init(zero_initials)
			s2.BeginSqueeze(r.rateSize, false)
			two := make([]byte, r.rateSize)
			s2.Squeeze(two[:half])
			s2.Squeeze(two[half:])

			if bytes.Equal(one, two) {
				t.Error("unaligned split unexpectedly matches one-shot")
			}
		})
	}
}

func TestSqueeze_StreamingVsNonStreaming_RateAligned(t *testing.T) {
	for _, r := range squeezeCases() {
		t.Run(r.name, func(t *testing.T) {
			var sc, su internal.State
			sc.Init(zero_initials)
			su.Init(zero_initials)
			sc.BeginSqueeze(r.rateSize, true)
			su.BeginSqueeze(r.rateSize, false)
			outC := make([]byte, r.rateSize)
			outU := make([]byte, r.rateSize)
			sc.Squeeze(outC)
			su.Squeeze(outU)
			if !bytes.Equal(outC, outU) {
				t.Error("streaming vs non-streaming mismatch")
			}
		})
	}
}
