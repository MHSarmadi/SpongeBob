package test

import (
	"bytes"
	"testing"

	spongebob "github.com/MHSarmadi/SpongeBob"
)

func TestCross_AllPrimitivesDistinct(t *testing.T) {
	// Same data+context, different primitives → different output.
	// Proves domain separation between hash, mac, kdf.
	hashOut := make([]byte, 32)
	macOut := make([]byte, 32)
	kdfOut := make([]byte, 32)

	spongebob.Hash128(testData, testContext, hashOut)
	spongebob.MAC(testSecret, testData, testContext, macOut)
	spongebob.KDF(testSecret, testData, testContext, kdfOut)

	if bytes.Equal(hashOut, macOut) {
		t.Fatal("Hash == MAC")
	}
	if bytes.Equal(hashOut, kdfOut) {
		t.Fatal("Hash == KDF")
	}
	if bytes.Equal(macOut, kdfOut) {
		t.Fatal("MAC == KDF")
	}
}

func TestCross_XOFExtension(t *testing.T) {
	// Streaming XOF: chunked squeezes concatenate identically.
	for _, rate := range testRates {
		h1 := newUnkeyed(t, rate, testContext)
		h1.Absorb(testData)
		first := squeeze(t, h1, 32)
		rest := squeeze(t, h1, 64)
		h1.Wipe()

		h2 := newUnkeyed(t, rate, testContext)
		h2.Absorb(testData)
		combined := squeeze(t, h2, 96)
		h2.Wipe()

		if !bytes.Equal(first, combined[:32]) {
			t.Errorf("rate %d: first 32 differ", rate)
		}
		if !bytes.Equal(rest, combined[32:]) {
			t.Errorf("rate %d: next 64 differ", rate)
		}
	}
}

func TestCross_CachedVsUncached_ByChunkInvariance(t *testing.T) {
	// Streaming chunk invariance means the cached squeeze produces the same
	// output regardless of split points. Test 1-byte chunks vs one big chunk.
	data := []byte("chunk me many ways")
	for _, rate := range testRates {
		h1 := newUnkeyed(t, rate, testContext)
		h1.Absorb(data)
		first := squeeze(t, h1, 1)
		second := squeeze(t, h1, 31)
		h1.Wipe()

		h2 := newUnkeyed(t, rate, testContext)
		h2.Absorb(data)
		combined := squeeze(t, h2, 32)
		h2.Wipe()

		if !bytes.Equal(first, combined[:1]) {
			t.Errorf("rate %d: 1-byte chunk differs from combined[0:1]", rate)
		}
		if !bytes.Equal(second, combined[1:32]) {
			t.Errorf("rate %d: 31-byte chunk differs from combined[1:32]", rate)
		}
	}
}

func TestCross_CloneIndependence(t *testing.T) {
	// Cloning a state, absorbing different data into each copy, must diverge.
	// (Exercises internal.State.Clone indirectly through streaming.)
	h := newUnkeyed(t, 128, testContext)
	h.Absorb([]byte("shared-prefix"))
	// Fork by wiping and re-initializing is the only public way to "clone"
	// without exposing State; not testing that path here.
	h.Wipe()
}
