package internal

import (
	"encoding/binary"
)

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func (s *State) absorb(data []byte, DOMAIN_SEPARATION uint64, RATE_BYTES int) {
	s.reset_cache(s.cache)

	rate_blocks := len(data)/RATE_BYTES + 1

	rate_block := s.state[:RATE_BYTES/8]

	for i := range rate_blocks {
		s.absorb_round(data[i*RATE_BYTES : min((i+1)*RATE_BYTES, len(data))])

		if i == rate_blocks-1 {
			pad(rate_block, RATE_BYTES-len(data)%RATE_BYTES)
		}

		s.state[15] ^= DOMAIN_SEPARATION

		s.state.Permute()
	}
}

func (s *State) absorb_round(bytes []byte) {
	i := 0
	for len(bytes) >= 8 {
		s.state[i] ^= binary.BigEndian.Uint64(bytes)
		bytes = bytes[8:]
		i++
	}
	if len(bytes) != 0 {
		var extra [8]byte
		copy(extra[:], bytes)
		s.state[i] ^= binary.BigEndian.Uint64(extra[:])
	}
}

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func (s *State) Absorb256(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP256_BYTES)
}

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func (s *State) Absorb512(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP512_BYTES)
}

// This function ALWAYS permute the state.
// Even for an empty `data`, it absorbs the padding which is never empty.
func (s *State) Absorb960(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP960_BYTES)
}
