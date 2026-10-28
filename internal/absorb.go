package internal

import (
	"encoding/binary"
)

// BeginAbsorb transitions _PHASE_READY → _PHASE_ABSORBING and configures the
// rate, streaming mode, and domain separation for the incoming message.
// Follow with one or more Absorb calls and (in streaming mode) SealAbsorb
// before BeginSqueeze.
func (s *State) BeginAbsorb(RATE_BYTES int, stream bool, DOMAIN_SEPARATION uint64) {
	switch RATE_BYTES {
	case RATE_OF_CAP256_BYTES, RATE_OF_CAP512_BYTES, RATE_OF_CAP960_BYTES:
	default:
		panic("Unknown rate bytes.")
	}
	switch s.phase {
	case _PHASE_FRESH:
		panic("Init first.")
	case _PHASE_READY:
		s.rate = RATE_BYTES
		s.phase = _PHASE_ABSORBING
		s.streaming = stream
		s.domain = DOMAIN_SEPARATION
		s.len = 0
	case _PHASE_ABSORBING:
		panic("Already absorbing.")
	case _PHASE_SQUEEZING:
		panic("Can't absorb mid-squeeze.")
	default:
		panic("Unknown phase.")
	}
}

// Absorb buffers data and absorbs each full rate-sized block into the state.
// stream=true:  calls accumulate; caller seals with SealAbsorb.
// stream=false: each call seals immediately (one Absorb per BeginAbsorb).
// Panics outside _PHASE_ABSORBING.
func (s *State) Absorb(data []byte) {
	switch s.phase {
	case _PHASE_FRESH:
		panic("Init first.")
	case _PHASE_READY:
		panic("BeginAbsorb first.")
	case _PHASE_ABSORBING:
		for len(data) > 0 {
			n := copy(s.buffer[s.len:], data)
			s.len += n
			data = data[n:]
			s.drain_full_blocks()
		}
		if !s.streaming {
			s.SealAbsorb()
		}
	case _PHASE_SQUEEZING:
		panic("Can't absorb mid-squeeze.")
	default:
		panic("Unknown phase.")
	}
}

// SealAbsorb pads the buffered tail with pad10*1 + rate domain, absorbs it
// as the final block, and transitions _PHASE_ABSORBING → _PHASE_READY.
// The single permute inside prepares the state for squeezing. Returns true
// on seal, false in _PHASE_READY (idempotent).
func (s *State) SealAbsorb() bool {
	switch s.phase {
	case _PHASE_FRESH:
		panic("Init first.")
	case _PHASE_READY:
		return false
	case _PHASE_ABSORBING:
		s.drain_last_block()
		s.phase = _PHASE_READY
		s.len = 0
		return true
	case _PHASE_SQUEEZING:
		panic("Can't seal absorption mid-squeeze.")
	default:
		panic("Unknown phase.")
	}
}

// drain_full_blocks absorbs every complete rate-sized block currently in
// the buffer, shifting remaining tail bytes to the front.
func (s *State) drain_full_blocks() {
	for s.len >= s.rate {
		s.absorb_block(s.buffer[:s.rate])
		copy(s.buffer[:], s.buffer[s.rate:s.len])
		s.len -= s.rate
	}
}

// drain_last_block pads the buffered tail via pad() and absorbs it as the
// final block of the message; state is left permuted.
func (s *State) drain_last_block() {
	rate_block := s.state[:s.rate/8]
	pad(rate_block, s.rate-s.len)
	s.absorb_block(s.buffer[:s.len])
}

// absorb_block XORs rate_block into state[0:rate/8], applies the domain
// separation to state[15], and permutes.
func (s *State) absorb_block(rate_block []byte) {
	i := 0
	for len(rate_block) >= 8 {
		s.state[i] ^= binary.BigEndian.Uint64(rate_block)
		rate_block = rate_block[8:]
		i++
	}
	if len(rate_block) != 0 {
		var extra [8]byte
		copy(extra[:], rate_block)
		s.state[i] ^= binary.BigEndian.Uint64(extra[:])
	}
	s.state[15] ^= s.domain
	s.state.Permute()
}
