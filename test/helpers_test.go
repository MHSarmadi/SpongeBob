package test

import (
	"testing"

	spongebob "github.com/MHSarmadi/SpongeBob"
	streambob "github.com/MHSarmadi/SpongeBob/StreamBob"
)

// squeezer is satisfied by both *streambob.Unkeyed and *streambob.Keyed,
// so squeeze-into-buffer tests are shared across the two hashers.
type squeezer interface {
	Squeeze(p []byte) (int, error)
}

// newUnkeyed builds an Unkeyed initialized for the given rate (128, 256, 480).
func newUnkeyed(t *testing.T, rate int, context string) *streambob.Unkeyed {
	t.Helper()
	h := &streambob.Unkeyed{}
	switch rate {
	case 128:
		h.Init128(context)
	case 256:
		h.Init256(context)
	case 480:
		h.Init480(context)
	default:
		t.Fatalf("bad rate %d", rate)
	}
	return h
}

// newKeyed is newUnkeyed but for the keyed hasher.
func newKeyed(t *testing.T, rate int, secret []byte, context string) *streambob.Keyed {
	t.Helper()
	h := &streambob.Keyed{}
	switch rate {
	case 128:
		h.Init128(secret, context)
	case 256:
		h.Init256(secret, context)
	case 480:
		h.Init480(secret, context)
	default:
		t.Fatalf("bad rate %d", rate)
	}
	return h
}

// squeeze pulls n bytes out of h via a fresh zeroed buffer.
func squeeze(t *testing.T, h squeezer, n int) []byte {
	t.Helper()
	out := make([]byte, n)
	if _, err := h.Squeeze(out); err != nil {
		t.Fatalf("squeeze: %v", err)
	}
	return out
}

// Fixtures shared across test files.
var (
	testRates   = []int{128, 256, 480}
	testContext = "spongebob-test"
	testData    = []byte("the quick brown fox jumps over the lazy dog")
	testSecret  = []byte("0123456789abcdef0123456789abcdef")
)

func newUninitUnkeyed() *streambob.Unkeyed { return &streambob.Unkeyed{} }
func newUninitKeyed() *streambob.Keyed     { return &streambob.Keyed{} }

func hashFn128(d []byte, c string, o []byte)    { spongebob.Hash128(d, c, o) }
func hashFn256(d []byte, c string, o []byte)    { spongebob.Hash256(d, c, o) }
func hashFn480(d []byte, c string, o []byte)    { spongebob.Hash480(d, c, o) }
func macDirect(k, d []byte, c string, o []byte) { spongebob.MAC(k, d, c, o) }

// modeName labels the streaming flag for subtest names.
func modeName(stream bool) string {
	if stream {
		return "streaming"
	}
	return "non-streaming"
}
