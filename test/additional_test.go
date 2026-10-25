package test

import (
	"bytes"
	"testing"
)

func TestUnkeyed_LargeDataMultiChunk(t *testing.T) {
	// At rate 96 (Variant128), Chunk(95) + Chunk(200) requires 3 permutes in StreamAbsorb.
	data := make([]byte, 95+200)
	for i := range data {
		data[i] = byte(i * 13)
	}
	one := oneShotHash(128, data, testContext)
	h := newUnkeyed(t, 128, testContext)
	h.Absorb(data[:95])
	h.Absorb(data[95:])
	streamed := squeeze(t, h, 32)
	h.Wipe()
	if !bytes.Equal(one, streamed) {
		t.Fatal("large multi-chunk mismatch — ab_len likely exceeded RATE_BYTES")
	}
}
