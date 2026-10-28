package test

import (
	"bytes"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
)

var zero_initials = [4]uint64{0, 0, 0, 0}

// absorbAndSqueeze builds a fresh state, absorbs data with the given rate and
// domain (non-streaming), then squeezes 32 bytes for observation.
func absorbAndSqueeze(rate int, data []byte, domain uint64) []byte {
	var s internal.State
	s.Init(zero_initials)
	s.BeginAbsorb(rate, false, domain)
	s.Absorb(data)
	s.BeginSqueeze(rate, false)
	out := make([]byte, 32)
	s.Squeeze(out)
	return out
}

type absorbCase struct {
	name string
	rate int
}

func absorbCases() []absorbCase {
	return []absorbCase{
		{"256", internal.RATE_OF_CAP256_BYTES},
		{"512", internal.RATE_OF_CAP512_BYTES},
		{"960", internal.RATE_OF_CAP960_BYTES},
	}
}

func TestAbsorb_Deterministic(t *testing.T) {
	data := []byte("hello world")
	for _, a := range absorbCases() {
		for _, domain := range []uint64{0, 1, 0xDEADBEEFCAFEBABE} {
			t.Run(a.name, func(t *testing.T) {
				x := absorbAndSqueeze(a.rate, data, domain)
				y := absorbAndSqueeze(a.rate, data, domain)
				if !bytes.Equal(x, y) {
					t.Errorf("domain=%#x: not deterministic", domain)
				}
			})
		}
	}
}

func TestAbsorb_DomainSeparation(t *testing.T) {
	data := []byte("hello")
	for _, a := range absorbCases() {
		t.Run(a.name, func(t *testing.T) {
			seen := make(map[[32]byte]uint64)
			for _, domain := range []uint64{0, 1, 2, 3, 0xDEADBEEFCAFEBABE, 0xFFFFFFFFFFFFFFFF} {
				out := absorbAndSqueeze(a.rate, data, domain)
				var key [32]byte
				copy(key[:], out)
				if prev, ok := seen[key]; ok {
					t.Errorf("domain=%#x collides with %#x", domain, prev)
				}
				seen[key] = domain
			}
		})
	}
}

func TestAbsorb_RateSensitivity(t *testing.T) {
	data := []byte("hello")
	seen := make(map[[32]byte]string)
	for _, a := range absorbCases() {
		out := absorbAndSqueeze(a.rate, data, 1)
		var key [32]byte
		copy(key[:], out)
		if prev, ok := seen[key]; ok {
			t.Errorf("rate %s collides with %s", a.name, prev)
		}
		seen[key] = a.name
	}
}

func TestAbsorb_EmptyInput(t *testing.T) {
	for _, a := range absorbCases() {
		for _, stream := range []bool{false, true} {
			t.Run(a.name+"/"+modeName(stream), func(t *testing.T) {
				var s internal.State
				s.Init(zero_initials)
				absorbOnce := func(data []byte, domain uint64) {
					s.BeginAbsorb(a.rate, stream, domain)
					s.Absorb(data)
					if stream {
						s.SealAbsorb()
					}
				}
				absorbOnce(nil, 0)
				absorbOnce([]byte{}, 0)
				absorbOnce([]byte{}, 0xDEADBEEF)
			})
		}
	}
}

func TestAbsorb_TrailingZerosDistinguished(t *testing.T) {
	cases := [][]byte{
		{}, {0}, {0, 0}, {0, 0, 0},
		{1}, {1, 0}, {1, 0, 0},
		{0xFF}, {0xFF, 0},
	}
	for _, a := range absorbCases() {
		t.Run(a.name, func(t *testing.T) {
			seen := make(map[[32]byte][]byte)
			for _, data := range cases {
				out := absorbAndSqueeze(a.rate, data, 0)
				var key [32]byte
				copy(key[:], out)
				if prev, ok := seen[key]; ok {
					t.Errorf("%v collides with %v", data, prev)
				}
				seen[key] = data
			}
		})
	}
}

func TestAbsorb_LengthBoundary(t *testing.T) {
	rates := []int{
		internal.RATE_OF_CAP256_BYTES,
		internal.RATE_OF_CAP512_BYTES,
		internal.RATE_OF_CAP960_BYTES,
	}
	for _, rate := range rates {
		for _, n := range []int{0, 1, rate - 1, rate, rate + 1, 2*rate - 1, 2 * rate, 2*rate + 1} {
			if n < 0 {
				continue
			}
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i * 7)
			}
			var s internal.State
			s.Init(zero_initials)
			s.BeginAbsorb(rate, false, 0xDEADBEEF)
			s.Absorb(data)
		}
	}
}
