package internal

import (
	"encoding/binary"
)

// StartAbsorb resets the absorb buffer for a fresh message. Call from Init.
func (s *State) StartAbsorb(ds uint64, enable_cache bool) {
	s.ab_len = 0
	s.ab_ds = ds
	s.ab_sealed = false
	s.reset_cache(enable_cache)
}

// StreamAbsorb feeds bytes without padding. Full blocks go straight to the
// permutation; the tail stays in ab_buf until Seal or the next call.
func (s *State) StreamAbsorb(data []byte, RATE_BYTES int) {
	if s.ab_sealed {
		panic("StreamAbsorb before StartAbsorb / after SealAbsorb")
	}
	if RATE_BYTES > len(s.ab_buf) {
		panic("StreamAbsorb/SealAbsorb: RATE_BYTES exceeds ab_buf")
	}
	if s.ab_len > 0 {
		n := copy(s.ab_buf[s.ab_len:], data)
		s.ab_len += n
		data = data[n:]
		if s.ab_len < RATE_BYTES {
			return
		}
		s.permuteBlock(s.ab_buf[:RATE_BYTES])
		s.ab_len = 0
	}
	for len(data) >= RATE_BYTES {
		s.permuteBlock(data[:RATE_BYTES])
		data = data[RATE_BYTES:]
	}
	s.ab_len = copy(s.ab_buf[:], data)
}

// SealAbsorb pads the buffered tail (with pad10*1 + rate domain) and permutes
// once. Idempotent — Squeeze calls this so a Write-then-Read works.
func (s *State) SealAbsorb(RATE_BYTES int) {
	if s.ab_sealed {
		return
	}
	if RATE_BYTES > len(s.ab_buf) {
		panic("StreamAbsorb/SealAbsorb: RATE_BYTES exceeds ab_buf")
	}
	rate_block := s.state[:RATE_BYTES/8]
	s.absorb_round(s.ab_buf[:s.ab_len])
	pad(rate_block, RATE_BYTES-s.ab_len)
	s.state[15] ^= s.ab_ds
	s.state.Permute()
	s.ab_len = 0
	s.ab_sealed = true
	s.reset_cache(s.cache)
}

func (s *State) permuteBlock(block []byte) {
	s.absorb_round(block)
	s.state[15] ^= s.ab_ds
	s.state.Permute()
	s.reset_cache(s.cache)
}

func (s *State) absorb(data []byte, DOMAIN_SEPARATION uint64, RATE_BYTES int) {
	if s.ab_sealed {
		s.StartAbsorb(DOMAIN_SEPARATION, s.cache)
	} else if s.ab_ds != DOMAIN_SEPARATION {
		panic("absorb: domain change mid-stream; call SealAbsorb first")
	}
	s.StreamAbsorb(data, RATE_BYTES)
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

func (s *State) Absorb256(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP256_BYTES)
}

func (s *State) Absorb512(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP512_BYTES)
}

func (s *State) Absorb960(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP960_BYTES)
}
