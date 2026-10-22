package internal

import (
	"encoding/binary"

	"github.com/MHSarmadi/Twister"
)

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func squeeze(state *Twister.State1024, out []byte, RATE_BYTES int) {
	rate_blocks := (len(out) + RATE_BYTES - 1) / RATE_BYTES // ceil(len / RATE_BYTES)

	rate_block := state[:RATE_BYTES/8]

	for i := range rate_blocks {
		squeeze_round(rate_block, out[i*RATE_BYTES:min((i+1)*RATE_BYTES, len(out))])

		state.Permute()
	}
}

func squeeze_round(words []uint64, bytes []byte) {
	i := 0
	var word_bytes [8]byte

	for len(bytes) > 0 {
		binary.BigEndian.PutUint64(word_bytes[:], words[i])
		remaining := min(8, len(bytes))

		for j := range remaining {
			bytes[j] ^= word_bytes[j]
		}

		bytes = bytes[remaining:]
		i++
	}
}

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func Squeeze256(state *Twister.State1024, out []byte) {
	squeeze(state, out, RATE_OF_CAP256_BYTES)
}

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func Squeeze512(state *Twister.State1024, out []byte) {
	squeeze(state, out, RATE_OF_CAP512_BYTES)
}

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func Squeeze960(state *Twister.State1024, out []byte) {
	squeeze(state, out, RATE_OF_CAP960_BYTES)
}
