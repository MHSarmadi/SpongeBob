package test

import (
	"bytes"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
)

func TestAbsorbThenSqueeze_Deterministic(t *testing.T) {
	msg := []byte("the quick brown fox")
	domain := uint64(0xAABBCCDDEEFF0011)

	run := func() []byte {
		var s internal.State
		s.Init(zero_initials)
		s.BeginAbsorb(internal.RATE_OF_CAP256_BYTES, false, domain)
		s.Absorb(msg)
		s.BeginSqueeze(internal.RATE_OF_CAP256_BYTES, false)
		out := make([]byte, 64)
		s.Squeeze(out)
		return out
	}

	if !bytes.Equal(run(), run()) {
		t.Fatal("not deterministic")
	}
}

func TestAbsorbThenSqueeze_StreamingMatchesNonStreaming(t *testing.T) {
	msg := []byte("the quick brown fox jumps over the lazy dog")
	for _, a := range absorbCases() {
		for _, r := range squeezeCases() {
			t.Run(a.name+"/"+r.name, func(t *testing.T) {
				// Non-streaming: absorb seals automatically.
				var sn internal.State
				sn.Init(zero_initials)
				sn.BeginAbsorb(a.rate, false, 0xABCDEF)
				sn.Absorb(msg)
				sn.BeginSqueeze(r.rateSize, false)
				outN := make([]byte, r.rateSize*2)
				sn.Squeeze(outN)

				// Streaming: explicit seal before BeginSqueeze.
				var ss internal.State
				ss.Init(zero_initials)
				ss.BeginAbsorb(a.rate, true, 0xABCDEF)
				ss.Absorb(msg)
				ss.SealAbsorb()
				ss.BeginSqueeze(r.rateSize, true)
				outS := make([]byte, r.rateSize*2)
				ss.Squeeze(outS)

				if !bytes.Equal(outN, outS) {
					t.Error("streaming vs non-streaming mismatch")
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
		var s internal.State
		s.Init(zero_initials)
		s.BeginAbsorb(internal.RATE_OF_CAP256_BYTES, false, 0)
		s.Absorb(msg)
		s.BeginSqueeze(internal.RATE_OF_CAP256_BYTES, false)
		out := make([]byte, 32)
		s.Squeeze(out)
		var key [32]byte
		copy(key[:], out)
		if prev, ok := seen[key]; ok {
			t.Errorf("msg %v collides with %v", msg, prev)
		}
		seen[key] = msg
	}
}

func TestAbsorbThenSqueeze_MultipleSqueezes(t *testing.T) {
	// Streaming chunked squeezes must equal one-shot streaming squeeze.
	setup := func() *internal.State {
		var s internal.State
		s.Init(zero_initials)
		s.BeginAbsorb(internal.RATE_OF_CAP256_BYTES, false, 0xDEAD)
		s.Absorb([]byte("msg"))
		s.BeginSqueeze(internal.RATE_OF_CAP256_BYTES, true)
		return &s
	}

	s1 := setup()
	stream := make([]byte, 96*3)
	for i := 0; i < 3; i++ {
		s1.Squeeze(stream[i*96 : (i+1)*96])
	}

	s2 := setup()
	oneShot := make([]byte, 96*3)
	s2.Squeeze(oneShot)

	if !bytes.Equal(stream, oneShot) {
		t.Error("multiple squeezes != one-shot")
	}
}
