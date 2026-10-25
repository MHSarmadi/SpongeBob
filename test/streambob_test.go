package test

import (
	"bytes"
	"testing"

	streambob "github.com/MHSarmadi/SpongeBob/StreamBob"
)

// ---------- Unkeyed ----------

func TestUnkeyed_ZeroValueErrors(t *testing.T) {
	var h struct{ u streambob.Unkeyed }
	_ = h
	u := newUninitUnkeyed()
	if _, err := u.Absorb([]byte("x")); err == nil {
		t.Error("Absorb on zero value did not error")
	}
	if _, err := u.Squeeze(make([]byte, 32)); err == nil {
		t.Error("Squeeze on zero value did not error")
	}
	if err := u.Permute(); err == nil {
		t.Error("Permute on zero value did not error")
	}
}

func TestUnkeyed_AfterWipeErrors(t *testing.T) {
	for _, rate := range testRates {
		h := newUnkeyed(t, rate, testContext)
		h.Absorb(testData)
		h.Wipe()
		if _, err := h.Absorb([]byte("x")); err == nil {
			t.Errorf("rate %d: Absorb after Wipe did not error", rate)
		}
		if _, err := h.Squeeze(make([]byte, 32)); err == nil {
			t.Errorf("rate %d: Squeeze after Wipe did not error", rate)
		}
	}
}

func TestUnkeyed_RateSeparation(t *testing.T) {
	outs := make(map[int][]byte)
	for _, rate := range testRates {
		h := newUnkeyed(t, rate, testContext)
		defer h.Wipe()
		h.Absorb(testData)
		outs[rate] = squeeze(t, h, 32)
	}
	if bytes.Equal(outs[128], outs[256]) || bytes.Equal(outs[128], outs[480]) || bytes.Equal(outs[256], outs[480]) {
		t.Fatal("rate variants collide")
	}
}

func TestUnkeyed_EmptyContextWorks(t *testing.T) {
	h := newUnkeyed(t, 128, "")
	defer h.Wipe()
	h.Absorb(testData)
	out := squeeze(t, h, 32)
	if bytes.Equal(out, make([]byte, 32)) {
		t.Fatal("empty context produced all-zero output")
	}
}

func TestUnkeyed_MatchesOneShot(t *testing.T) {
	data := []byte("the quick brown fox jumps over the lazy dog")
	chunkings := [][]int{
		{len(data)},
		{1, len(data) - 1},
		{7, 11, len(data) - 18},
	}
	byteByByte := make([]int, len(data))
	for i := range byteByByte {
		byteByByte[i] = 1
	}
	chunkings = append(chunkings, byteByByte)

	for _, rate := range testRates {
		for ci, chunks := range chunkings {
			one := oneShotHash(rate, data, testContext)
			h := newUnkeyed(t, rate, testContext)
			off := 0
			for _, n := range chunks {
				end := off + n
				if end > len(data) {
					end = len(data)
				}
				h.Absorb(data[off:end])
				off = end
			}
			streamed := squeeze(t, h, 32)
			h.Wipe()
			if !bytes.Equal(one, streamed) {
				t.Errorf("rate %d chunking %d: one-shot != streaming", rate, ci)
			}
		}
	}
}

func TestUnkeyed_PermuteActsAsSeparator(t *testing.T) {
	sep := newUnkeyed(t, 128, testContext)
	defer sep.Wipe()
	sep.Absorb([]byte("aaa"))
	sep.Permute()
	sep.Absorb([]byte("bbb"))
	outSep := squeeze(t, sep, 32)

	joined := newUnkeyed(t, 128, testContext)
	defer joined.Wipe()
	joined.Absorb([]byte("aaabbb"))
	outJoined := squeeze(t, joined, 32)

	if bytes.Equal(outSep, outJoined) {
		t.Fatal("Permute between absorptions made no difference")
	}
}

func TestUnkeyed_PermuteBeforeAbsorbNoop(t *testing.T) {
	h1 := newUnkeyed(t, 128, testContext)
	defer h1.Wipe()
	h1.Permute()
	out1 := squeeze(t, h1, 32)

	h2 := newUnkeyed(t, 128, testContext)
	defer h2.Wipe()
	out2 := squeeze(t, h2, 32)

	if bytes.Equal(out1, out2) {
		t.Fatal("Permutation didn't happen.")
	}
}

// ---------- Keyed ----------

func TestKeyed_ZeroValueErrors(t *testing.T) {
	k := newUninitKeyed()
	if _, err := k.Absorb([]byte("x")); err == nil {
		t.Error("Absorb on zero value did not error")
	}
	if _, err := k.Squeeze(make([]byte, 32)); err == nil {
		t.Error("Squeeze on zero value did not error")
	}
	if err := k.Permute(); err == nil {
		t.Error("Permute on zero value did not error")
	}
}

func TestKeyed_AfterWipeErrors(t *testing.T) {
	for _, rate := range testRates {
		h := newKeyed(t, rate, testSecret, testContext)
		h.Absorb(testData)
		h.Wipe()
		if _, err := h.Absorb([]byte("x")); err == nil {
			t.Errorf("rate %d: Absorb after Wipe did not error", rate)
		}
	}
}

func TestKeyed_SecretSensitivity(t *testing.T) {
	secrets := [][]byte{
		{0}, {1},
		[]byte("key"),
		testSecret,
		bytes.Repeat([]byte{0xFF}, 32),
	}
	seen := make(map[string][]byte)
	for _, sk := range secrets {
		h := newKeyed(t, 128, sk, testContext)
		h.Absorb(testData)
		out := squeeze(t, h, 32)
		h.Wipe()
		if prev, ok := seen[string(out)]; ok {
			t.Errorf("secret %q collides with %q", sk, prev)
		}
		seen[string(out)] = sk
	}
}

func TestKeyed_KeyedDiffersFromUnkeyed(t *testing.T) {
	u := newUnkeyed(t, 128, testContext)
	defer u.Wipe()
	u.Absorb(testData)
	outU := squeeze(t, u, 32)

	k := newKeyed(t, 128, testSecret, testContext)
	defer k.Wipe()
	k.Absorb(testData)
	outK := squeeze(t, k, 32)

	if bytes.Equal(outU, outK) {
		t.Fatal("keyed and unkeyed produce same output")
	}
}

func TestKeyed_MatchesMAC(t *testing.T) {
	// MAC is exactly Keyed.Init128 + Absorb + Squeeze.
	mac := make([]byte, 32)
	macDirect(testSecret, testData, testContext, mac)

	k := newKeyed(t, 128, testSecret, testContext)
	defer k.Wipe()
	k.Absorb(testData)
	streamed := squeeze(t, k, 32)

	if !bytes.Equal(mac, streamed) {
		t.Fatal("MAC != Keyed streaming equivalent")
	}
}

// ---------- helpers specific to this file ----------

func oneShotHash(rate int, data []byte, ctx string) []byte {
	out := make([]byte, 32)
	switch rate {
	case 128:
		hashFn128(data, ctx, out)
	case 256:
		hashFn256(data, ctx, out)
	case 480:
		hashFn480(data, ctx, out)
	}
	return out
}
