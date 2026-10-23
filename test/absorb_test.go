package test

import (
	"bytes"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
	"github.com/MHSarmadi/Twister"
)

// absorbAndSqueeze absorbs `data` with the given absorb method + domain, then
// squeezes 32 bytes to observe the resulting state. The squeeze rate is
// arbitrary; we only need a consistent observation function.
func absorbAndSqueeze(absorb func(*internal.State, []byte, uint64), data []byte, domain uint64) []byte {
	s := internal.State{}
	s.Reset(false)
	absorb(&s, data, domain)
	out := make([]byte, 32)
	s.Squeeze256(out)
	return out
}

type absorbCase struct {
	name string
	f    func(*internal.State, []byte, uint64)
}

func absorbCases() []absorbCase {
	return []absorbCase{
		{"256", (*internal.State).Absorb256},
		{"512", (*internal.State).Absorb512},
		{"960", (*internal.State).Absorb960},
	}
}

func TestAbsorb_Deterministic(t *testing.T) {
	data := []byte("hello world")
	for _, a := range absorbCases() {
		for _, domain := range []uint64{0, 1, 0xDEADBEEFCAFEBABE} {
			t.Run(a.name, func(t *testing.T) {
				x := absorbAndSqueeze(a.f, data, domain)
				y := absorbAndSqueeze(a.f, data, domain)
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
				out := absorbAndSqueeze(a.f, data, domain)
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
		out := absorbAndSqueeze(a.f, data, 1)
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
		s := internal.State{}
		s.Reset(false)
		// Must not panic.
		a.f(&s, nil, 0)
		a.f(&s, []byte{}, 0)
		a.f(&s, []byte{}, 0xDEADBEEF)
	}
}

func TestAbsorb_TrailingZerosDistinguished(t *testing.T) {
	// pad10*1 must distinguish inputs that differ only in trailing zeros.
	cases := [][]byte{
		{}, {0}, {0, 0}, {0, 0, 0},
		{1}, {1, 0}, {1, 0, 0},
		{0xFF}, {0xFF, 0},
	}
	for _, a := range absorbCases() {
		t.Run(a.name, func(t *testing.T) {
			seen := make(map[[32]byte][]byte)
			for _, data := range cases {
				out := absorbAndSqueeze(a.f, data, 0)
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
	// Lengths around block boundaries must not panic.
	type rateCase struct {
		n int
		f func(*internal.State, []byte, uint64)
	}
	rates := []rateCase{
		{internal.RATE_OF_CAP256_BYTES, (*internal.State).Absorb256},
		{internal.RATE_OF_CAP512_BYTES, (*internal.State).Absorb512},
		{internal.RATE_OF_CAP960_BYTES, (*internal.State).Absorb960},
	}
	for _, r := range rates {
		for _, n := range []int{0, 1, r.n - 1, r.n, r.n + 1, 2*r.n - 1, 2 * r.n, 2*r.n + 1} {
			if n < 0 {
				continue
			}
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i * 7)
			}
			s := internal.State{}
			s.Reset(false)
			r.f(&s, data, 0xDEADBEEF)
		}
	}
}

func TestAbsorb_ResetsCache(t *testing.T) {
	for _, a := range absorbCases() {
		t.Run(a.name, func(t *testing.T) {
			// Build a state, prime the cache via a short squeeze.
			var st Twister.State1024
			sA := internal.State{}
			sA.ResetFromExisting(&st, true)
			a.f(&sA, []byte("seed"), 0)
			sA.Squeeze256(make([]byte, 4)) // cache now holds ~92 bytes

			// Fork: both paths share the same state from here on.
			snapshot := st

			// Path A: primed cache, absorb, squeeze.
			a.f(&sA, []byte("x"), 0xABC)
			outA := make([]byte, 32)
			sA.Squeeze256(outA)

			// Path B: same state, fresh (empty) cache, absorb, squeeze.
			var stB Twister.State1024
			stB = snapshot
			sB := internal.State{}
			sB.ResetFromExisting(&stB, true)
			a.f(&sB, []byte("x"), 0xABC)
			outB := make([]byte, 32)
			sB.Squeeze256(outB)

			if !bytes.Equal(outA, outB) {
				t.Error("cache not reset by absorb")
			}
		})
	}
}
