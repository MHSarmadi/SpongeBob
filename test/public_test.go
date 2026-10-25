package test

import (
	"bytes"
	"testing"

	spongebob "github.com/MHSarmadi/SpongeBob"
)

// ---------- Hash: determinism & sensitivity ----------

func TestHash_Deterministic(t *testing.T) {
	fns := []struct {
		name string
		fn   func([]byte, string, []byte)
	}{
		{"Hash128", spongebob.Hash128},
		{"Hash256", spongebob.Hash256},
		{"Hash480", spongebob.Hash480},
	}
	for _, tc := range fns {
		t.Run(tc.name, func(t *testing.T) {
			a := make([]byte, 32)
			b := make([]byte, 32)
			tc.fn(testData, testContext, a)
			tc.fn(testData, testContext, b)
			if !bytes.Equal(a, b) {
				t.Fatal("not deterministic")
			}
		})
	}
}

func TestHash_DataSensitivity(t *testing.T) {
	inputs := [][]byte{
		{}, {0}, {1},
		[]byte("a"), []byte("b"),
		[]byte("ab"), []byte("ba"),
		testData,
		bytes.Repeat([]byte{0xAA}, 96),
		bytes.Repeat([]byte{0xAA}, 97),
	}
	fns := []struct {
		name string
		fn   func([]byte, string, []byte)
	}{
		{"Hash128", spongebob.Hash128},
		{"Hash256", spongebob.Hash256},
		{"Hash480", spongebob.Hash480},
	}
	for _, tc := range fns {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(map[string][]byte)
			for _, in := range inputs {
				out := make([]byte, 32)
				tc.fn(in, testContext, out)
				if prev, ok := seen[string(out)]; ok {
					t.Errorf("%q collides with %q", in, prev)
				}
				seen[string(out)] = in
			}
		})
	}
}

func TestHash_ContextSensitivity(t *testing.T) {
	contexts := []string{"", "a", "b", "ctx", testContext, "some-other-context"}
	fns := []struct {
		name string
		fn   func([]byte, string, []byte)
	}{
		{"Hash128", spongebob.Hash128},
		{"Hash256", spongebob.Hash256},
		{"Hash480", spongebob.Hash480},
	}
	for _, tc := range fns {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(map[string]string)
			for _, ctx := range contexts {
				out := make([]byte, 32)
				tc.fn(testData, ctx, out)
				if prev, ok := seen[string(out)]; ok {
					t.Errorf("context %q collides with %q", ctx, prev)
				}
				seen[string(out)] = ctx
			}
		})
	}
}

func TestHash_NilAndEmptyEquivalent(t *testing.T) {
	fns := []func([]byte, string, []byte){
		spongebob.Hash128, spongebob.Hash256, spongebob.Hash480,
	}
	for i, fn := range fns {
		a := make([]byte, 32)
		b := make([]byte, 32)
		fn(nil, testContext, a)
		fn([]byte{}, testContext, b)
		if !bytes.Equal(a, b) {
			t.Errorf("fn %d: nil vs empty []byte differ", i)
		}
	}
}

func TestHash_EmptyVsZeroByteDiffer(t *testing.T) {
	empty := make([]byte, 32)
	zero := make([]byte, 32)
	spongebob.Hash128(nil, testContext, empty)
	spongebob.Hash128([]byte{0}, testContext, zero)
	if bytes.Equal(empty, zero) {
		t.Fatal("empty and {0} collide")
	}
}

func TestHash_RateSeparation(t *testing.T) {
	a := make([]byte, 32)
	b := make([]byte, 32)
	c := make([]byte, 32)
	spongebob.Hash128(testData, testContext, a)
	spongebob.Hash256(testData, testContext, b)
	spongebob.Hash480(testData, testContext, c)
	if bytes.Equal(a, b) || bytes.Equal(a, c) || bytes.Equal(b, c) {
		t.Fatal("rate variants collide")
	}
}

func TestHash_NotEmpty(t *testing.T) {
	fns := []struct {
		name string
		fn   func([]byte, string, []byte)
	}{
		{"Hash128", spongebob.Hash128},
		{"Hash256", spongebob.Hash256},
		{"Hash480", spongebob.Hash480},
	}
	for _, tc := range fns {
		t.Run(tc.name, func(t *testing.T) {
			out := make([]byte, 32)
			tc.fn(testData, testContext, out)
			if bytes.Equal(out, make([]byte, 32)) {
				t.Fatal("all-zero output")
			}
		})
	}
}

// ---------- MAC ----------

func TestMAC_Deterministic(t *testing.T) {
	a := make([]byte, 32)
	b := make([]byte, 32)
	spongebob.MAC(testSecret, testData, testContext, a)
	spongebob.MAC(testSecret, testData, testContext, b)
	if !bytes.Equal(a, b) {
		t.Fatal("not deterministic")
	}
}

func TestMAC_SecretSensitivity(t *testing.T) {
	secrets := [][]byte{
		{0}, {1},
		[]byte("key"), []byte("kex"),
		testSecret,
		bytes.Repeat([]byte{0xFF}, 32),
	}
	seen := make(map[string][]byte)
	for _, sk := range secrets {
		out := make([]byte, 32)
		spongebob.MAC(sk, testData, testContext, out)
		if prev, ok := seen[string(out)]; ok {
			t.Errorf("secret %q collides with %q", sk, prev)
		}
		seen[string(out)] = sk
	}
}

func TestMAC_DataSensitivity(t *testing.T) {
	data := [][]byte{
		nil, {0}, {1},
		[]byte("msg"), []byte("msh"),
		testData,
	}
	seen := make(map[string][]byte)
	for _, d := range data {
		out := make([]byte, 32)
		spongebob.MAC(testSecret, d, testContext, out)
		if prev, ok := seen[string(out)]; ok {
			t.Errorf("data %q collides with %q", d, prev)
		}
		seen[string(out)] = d
	}
}

func TestMAC_ContextSensitivity(t *testing.T) {
	contexts := []string{"", "a", "b", testContext, "another"}
	seen := make(map[string]string)
	for _, ctx := range contexts {
		out := make([]byte, 32)
		spongebob.MAC(testSecret, testData, ctx, out)
		if prev, ok := seen[string(out)]; ok {
			t.Errorf("context %q collides with %q", ctx, prev)
		}
		seen[string(out)] = ctx
	}
}

func TestMAC_EmptySecretWorks(t *testing.T) {
	// Empty secret must not panic; must produce deterministic non-zero output.
	a := make([]byte, 32)
	b := make([]byte, 32)
	spongebob.MAC(nil, testData, testContext, a)
	spongebob.MAC([]byte{}, testData, testContext, b)
	if !bytes.Equal(a, b) {
		t.Error("nil vs empty secret differ")
	}
	if bytes.Equal(a, make([]byte, 32)) {
		t.Fatal("empty secret produced all-zero output")
	}
}

func TestMAC_DiffersFromHash(t *testing.T) {
	mac := make([]byte, 32)
	hash := make([]byte, 32)
	spongebob.MAC(testSecret, testData, testContext, mac)
	// Hash of secret||data with the same context would be the naive (wrong) MAC.
	concatenated := append(append([]byte{}, testSecret...), testData...)
	spongebob.Hash128(concatenated, testContext, hash)
	if bytes.Equal(mac, hash) {
		t.Fatal("MAC == Hash(secret||data) — no domain separation")
	}
}

// ---------- KDF ----------

func TestKDF_Deterministic(t *testing.T) {
	a := make([]byte, 32)
	b := make([]byte, 32)
	spongebob.KDF(testSecret, []byte("salt"), testContext, a)
	spongebob.KDF(testSecret, []byte("salt"), testContext, b)
	if !bytes.Equal(a, b) {
		t.Fatal("not deterministic")
	}
}

func TestKDF_MasterSensitivity(t *testing.T) {
	masters := [][]byte{
		nil, {0}, {1},
		[]byte("m1"), []byte("m2"),
		testSecret,
		bytes.Repeat([]byte{0xFF}, 32),
	}
	seen := make(map[string][]byte)
	for _, m := range masters {
		out := make([]byte, 32)
		spongebob.KDF(m, []byte("salt"), testContext, out)
		if prev, ok := seen[string(out)]; ok {
			// nil and empty are equivalent
			if !(len(m) == 0 && len(prev) == 0) {
				t.Errorf("master %q collides with %q", m, prev)
			}
		}
		seen[string(out)] = m
	}
}

func TestKDF_SaltSensitivity(t *testing.T) {
	salts := [][]byte{
		nil, {0}, {1},
		[]byte("s1"), []byte("s2"),
		bytes.Repeat([]byte{0x55}, 32),
	}
	seen := make(map[string][]byte)
	for _, s := range salts {
		out := make([]byte, 32)
		spongebob.KDF(testSecret, s, testContext, out)
		if prev, ok := seen[string(out)]; ok {
			if !(len(s) == 0 && len(prev) == 0) {
				t.Errorf("salt %q collides with %q", s, prev)
			}
		}
		seen[string(out)] = s
	}
}

func TestKDF_ContextSensitivity(t *testing.T) {
	contexts := []string{"", "a", "b", testContext, "another"}
	seen := make(map[string]string)
	for _, c := range contexts {
		out := make([]byte, 32)
		spongebob.KDF(testSecret, []byte("salt"), c, out)
		if prev, ok := seen[string(out)]; ok {
			t.Errorf("context %q collides with %q", c, prev)
		}
		seen[string(out)] = c
	}
}

func TestKDF_DiffersFromHash(t *testing.T) {
	kdf := make([]byte, 32)
	spongebob.KDF(testSecret, []byte("salt"), testContext, kdf)

	candidates := [][]byte{
		testSecret, // hash of master alone
		append(append([]byte{}, []byte("salt")...), testSecret...), // salt||master
		append(append([]byte{}, testSecret...), []byte("salt")...), // master||salt
	}
	for i, c := range candidates {
		h := make([]byte, 32)
		spongebob.Hash128(c, testContext, h)
		if bytes.Equal(kdf, h) {
			t.Errorf("KDF == Hash of candidate %d", i)
		}
	}
}

func TestKDF_EmptyInputsWork(t *testing.T) {
	cases := []struct {
		name         string
		master, salt []byte
	}{
		{"nil/nil", nil, nil},
		{"nil/salt", nil, []byte("salt")},
		{"master/nil", testSecret, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := make([]byte, 32)
			spongebob.KDF(c.master, c.salt, testContext, out)
			if bytes.Equal(out, make([]byte, 32)) {
				t.Fatal("all-zero output")
			}
		})
	}
}
