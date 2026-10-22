package internal

import (
	"encoding/binary"

	"github.com/MHSarmadi/Twister"
)

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func absorb(state *Twister.State1024, data []byte, DOMAIN_SEPARATION uint64, RATE_BYTES int) {
	rate_blocks := len(data)/RATE_BYTES + 1

	rate_block := state[:RATE_BYTES/8]

	for i := range rate_blocks {
		absorb_round(rate_block, data[i*RATE_BYTES:min((i+1)*RATE_BYTES, len(data))])

		if i == rate_blocks-1 {
			pad(rate_block, RATE_BYTES-len(data)%RATE_BYTES)
		}

		state[15] ^= DOMAIN_SEPARATION

		state.Permute()
	}
}

func absorb_round(words []uint64, bytes []byte) {
	i := 0
	for len(bytes) >= 8 {
		words[i] ^= binary.BigEndian.Uint64(bytes)
		bytes = bytes[8:]
		i++
	}
	if len(bytes) != 0 {
		var extra [8]byte
		copy(extra[:], bytes)
		words[i] ^= binary.BigEndian.Uint64(extra[:])
	}
}

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func Absorb256(state *Twister.State1024, data []byte, domain_separation uint64) {
	absorb(state, data, domain_separation, RATE_OF_CAP256_BYTES)
}

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func Absorb512(state *Twister.State1024, data []byte, domain_separation uint64) {
	absorb(state, data, domain_separation, RATE_OF_CAP512_BYTES)
}

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func Absorb960(state *Twister.State1024, data []byte, domain_separation uint64) {
	absorb(state, data, domain_separation, RATE_OF_CAP960_BYTES)
}
