package test

import (
	"bytes"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
)

func TestAbsorbThenSqueeze_Deterministic(t *testing.T) {
	msg := []byte("the quick brown fox")
	domain := uint64(0xAABBCCDDEEFF0011)

	s1 := internal.State{}
	s1.Reset(false)
	s2 := internal.State{}
	s2.Reset(false)
	s1.Absorb256(msg, domain)
	s2.Absorb256(msg, domain)

	out1 := make([]byte, 64)
	out2 := make([]byte, 64)
	s1.Squeeze256(out1)
	s2.Squeeze256(out2)
	if !bytes.Equal(out1, out2) {
		t.Fatal("not deterministic")
	}
}

func TestAbsorbThenSqueeze_CachedMatchesUncached(t *testing.T) {
	msg := []byte("the quick brown fox jumps over the lazy dog")
	for _, a := range absorbCases() {
		for _, r := range squeezeCases() {
			t.Run(a.name+"/"+r.name, func(t *testing.T) {
				sc := internal.State{}
				sc.Reset(true)
				su := internal.State{}
				su.Reset(false)
				a.f(&sc, msg, 0xABCDEF)
				a.f(&su, msg, 0xABCDEF)

				outC := make([]byte, r.rateSize*2)
				outU := make([]byte, r.rateSize*2)
				r.fn(&sc, outC)
				r.fn(&su, outU)
				if !bytes.Equal(outC, outU) {
					t.Error("cached vs uncached mismatch after absorb")
				}
			})
		}
	}
}

func TestAbsorbThenSqueeze_MessageSensitivity(t *testing.T) {
	msgs := [][]byte{
		{}, {0}, {1},
		[]byte("a"), []byte("b"),
		[]byte("ab"), []byte("ba"),
		bytes.Repeat([]byte{0xAA}, 96),
		bytes.Repeat([]byte{0xAA}, 97),
	}
	seen := make(map[[32]byte][]byte)
	for _, msg := range msgs {
		s := internal.State{}
		s.Reset(false)
		s.Absorb256(msg, 0)
		out := make([]byte, 32)
		s.Squeeze256(out)
		var key [32]byte
		copy(key[:], out)
		if prev, ok := seen[key]; ok {
			t.Errorf("msg %v collides with %v", msg, prev)
		}
		seen[key] = msg
	}
}

func TestAbsorbThenSqueeze_MultipleSqueezes(t *testing.T) {
	// Consecutive cached squeezes must produce a continuous stream.
	s1 := internal.State{}
	s1.Reset(true)
	s1.Absorb256([]byte("msg"), 0xDEAD)
	stream := make([]byte, 96*3)
	for i := 0; i < 3; i++ {
		s1.Squeeze256(stream[i*96 : (i+1)*96])
	}

	s2 := internal.State{}
	s2.Reset(true)
	s2.Absorb256([]byte("msg"), 0xDEAD)
	oneShot := make([]byte, 96*3)
	s2.Squeeze256(oneShot)

	if !bytes.Equal(stream, oneShot) {
		t.Error("multiple squeezes != one-shot")
	}
}
