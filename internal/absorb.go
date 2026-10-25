package internal

import (
	"encoding/binary"
)

// StartAbsorb resets the absorb buffer for a fresh message. Call from Init.
func (s *State) StartAbsorb(ds uint64, rate int, enable_cache bool) {
	s.ab_len = 0
	s.ab_ds = ds
	s.ab_rate = rate
	s.ab_sealed = false
	s.reset_cache(enable_cache)
}

// StreamAbsorb feeds bytes without padding. Full blocks go straight to the
// permutation; the tail stays in ab_buf until Seal or the next call.
func (s *State) StreamAbsorb(data []byte, RATE_BYTES int) {
	if s.ab_sealed {
		panic("StreamAbsorb before StartAbsorb / after SealAbsorb")
	}
	if RATE_BYTES != s.ab_rate {
		panic("StreamAbsorb: RATE_BYTES does not match StartAbsorb rate")
	}
	if RATE_BYTES > len(s.ab_buf) {
		panic("StreamAbsorb: RATE_BYTES exceeds ab_buf")
	}

	for len(data) > 0 {
		n := copy(s.ab_buf[s.ab_len:], data)
		s.ab_len += n
		data = data[n:]

		// Drain EVERY full block currently buffered.
		for s.ab_len >= RATE_BYTES {
			s.permuteBlock(s.ab_buf[:RATE_BYTES])
			copy(s.ab_buf[:], s.ab_buf[RATE_BYTES:s.ab_len])
			s.ab_len -= RATE_BYTES
		}
	}
}

// SealAbsorb pads the buffered tail (with pad10*1 + rate domain) and permutes
// once. Idempotent — Squeeze calls this so a Write-then-Read works.
// returns true if any Sealing has happened.
func (s *State) SealAbsorb() bool {
	if s.ab_sealed {
		return false
	}
	if s.ab_rate == 0 || s.ab_rate > len(s.ab_buf) {
		panic("SealAbsorb: ab_rate not set / out of range")
	}
	rate_block := s.state[:s.ab_rate/8]
	s.absorb_round(s.ab_buf[:s.ab_len])
	pad(rate_block, s.ab_rate-s.ab_len)
	s.state[15] ^= s.ab_ds
	s.state.Permute()
	s.ab_len = 0
	s.ab_sealed = true
	s.reset_cache(s.streaming)
	return true
}

func (s *State) permuteBlock(block []byte) {
	s.absorb_round(block)
	s.state[15] ^= s.ab_ds
	s.state.Permute()
	s.reset_cache(s.streaming)
}

func (s *State) absorb(data []byte, ds uint64, RATE_BYTES int) {
	if s.ab_sealed {
		s.StartAbsorb(ds, RATE_BYTES, s.streaming)
	} else if s.ab_ds != ds || s.ab_rate != RATE_BYTES {
		panic("absorb: domain or rate change mid-stream; call SealAbsorb first")
	}
	s.StreamAbsorb(data, RATE_BYTES)
	if !s.streaming {
		s.SealAbsorb()
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

func (s *State) Absorb256(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP256_BYTES)
}

func (s *State) Absorb512(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP512_BYTES)
}

func (s *State) Absorb960(data []byte, domain_separation uint64) {
	s.absorb(data, domain_separation, RATE_OF_CAP960_BYTES)
}
