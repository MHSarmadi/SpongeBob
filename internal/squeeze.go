package internal

import (
	"encoding/binary"
)

// BeginSqueeze transitions _PHASE_READY → _PHASE_SQUEEZING and configures the
// output rate and streaming mode. Streaming callers must call SealAbsorb
// before this — BeginSqueeze does not seal implicitly (panics in
// _PHASE_ABSORBING).
func (s *State) BeginSqueeze(RATE_BYTES int, stream bool) {
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
		s.phase = _PHASE_SQUEEZING
		s.streaming = stream
		s.len = 0
	case _PHASE_ABSORBING:
		panic("Can't squeeze mid-absorption.")
	case _PHASE_SQUEEZING:
		panic("Already squeezing.")
	default:
		panic("Unknown phase.")
	}
}

// Squeeze XORs the output stream into out. Empty out is a no-op.
// stream=false: out must be read in rate-sized chunks; each block permutes.
// stream=true:  arbitrary chunk sizes; pending bytes are shifted, not dropped.
// Panics unless _PHASE_SQUEEZING.
func (s *State) Squeeze(out []byte) {
	switch s.phase {
	case _PHASE_FRESH:
		panic("Init first.")
	case _PHASE_READY:
		panic("BeginSqueeze first.")
	case _PHASE_ABSORBING:
		panic("Can't squeeze mid-absorption.")
	case _PHASE_SQUEEZING:
		if s.streaming {
			got := 0
			for got < len(out) {
				if s.len == 0 {
					s.squeeze_block_stream()
					s.state.Permute()
				}
				take := min(len(out)-got, s.len)

				for j := range take {
					out[got+j] ^= s.buffer[j]
				}

				copy(s.buffer[:], s.buffer[take:s.len])
				s.len -= take
				got += take
			}
		} else {
			rate_blocks := (len(out) + s.rate - 1) / s.rate

			for i := range rate_blocks {
				s.squeeze_block_nonstream(out[i*s.rate : min((i+1)*s.rate, len(out))])
				s.state.Permute()
			}
		}
	default:
		panic("Unknown phase.")
	}
}

// squeeze_block_nonstream writes up to one rate-block of the current state
// into bytes (XOR). Caller is responsible for the permute.
func (s *State) squeeze_block_nonstream(bytes []byte) {
	i := 0
	var word_bytes [8]byte

	for len(bytes) > 0 {
		binary.BigEndian.PutUint64(word_bytes[:], s.state[i])
		remaining := min(8, len(bytes))

		for j := range remaining {
			bytes[j] ^= word_bytes[j]
		}

		bytes = bytes[remaining:]
		i++
	}
}

// squeeze_block_stream copies exactly s.rate bytes of the current state into
// s.buffer and sets s.len = s.rate. Caller permutes after calling.
func (s *State) squeeze_block_stream() {
	var word_bytes [8]byte
	i, written, remaining := 0, 0, s.rate
	for remaining > 0 {
		binary.BigEndian.PutUint64(word_bytes[:], s.state[i])
		chunk := min(8, remaining)
		copy(s.buffer[written:], word_bytes[:chunk])
		written += chunk
		remaining -= chunk
		i++
	}
	s.len = s.rate
}
