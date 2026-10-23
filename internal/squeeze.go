package internal

import (
	"encoding/binary"
)

// This function squeezes into XOR. Pass a full-zero `out` if you want pure
// extraction. Empty `out` does nothing — state is not permuted.
//
// With cache enabled, arbitrary chunk boundaries produce the same concatenated
// output as one big call. Without cache, chunking must align to RATE_BYTES.
func (s *State) squeeze(out []byte, RATE_BYTES int) {
	if s.cache {
		s.squeeze_cached(out, RATE_BYTES)
	} else {
		s.squeeze_uncached(out, RATE_BYTES)
	}
}

func (s *State) squeeze_uncached(out []byte, RATE_BYTES int) {
	rate_blocks := (len(out) + RATE_BYTES - 1) / RATE_BYTES

	for i := range rate_blocks {
		s.squeeze_round_uncached(out[i*RATE_BYTES : min((i+1)*RATE_BYTES, len(out))])
		s.state.Permute()
	}
}

// squeeze_cached extracts from state through the ring buffer, so output is
// independent of how the caller chunks `out`. Refill = squeeze_roundCached
// followed by Permute, matching squeezeUncached per-block order exactly.
//
// The inner XOR loop is split at the ring wrap boundary, so each byte does a
// plain indexed access instead of a modulo. The second range starts at buf[0]
// because after consuming `first` bytes the read cursor is exactly at the wrap
// point by construction.
func (s *State) squeeze_cached(out []byte, RATE_BYTES int) {
	got := 0
	for got < len(out) {
		if s.n == 0 {
			s.squeeze_round_cached(RATE_BYTES)
			s.state.Permute()
		}
		take := min(len(out)-got, s.n)

		first := min(take, len(s.buf)-s.pos) // bytes before the ring wraps
		for j := range first {
			out[got+j] ^= s.buf[s.pos+j]
		}
		// take <= RATE_BYTES <= len(s.buf), so the tail wraps at most once
		// and `take-first` never exceeds the buffer.
		for j := range take - first {
			out[got+first+j] ^= s.buf[j]
		}

		s.pos = (s.pos + take) % len(s.buf)
		s.n -= take
		got += take
	}
}

// squeeze_round_uncached XORs bytes of the current rate into `bytes` (uncached path).
func (s *State) squeeze_round_uncached(bytes []byte) {
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

// squeeze_round_cached extracts RATE_BYTES from state into s.buf at s.pos,
// wrapping via %128. Assumes s.n == 0 (buffer empty), so overwrites are safe.
// Does NOT permute; the caller permutes after refill.
func (s *State) squeeze_round_cached(RATE_BYTES int) {
	wp := uint(s.pos)
	var word_bytes [8]byte
	i := 0
	remaining := RATE_BYTES
	for remaining > 0 {
		binary.BigEndian.PutUint64(word_bytes[:], s.state[i])
		chunk := uint(min(8, remaining))
		for j := range chunk {
			s.buf[(wp+j)&127] = word_bytes[j]
		}
		wp = (wp + chunk) & 127
		remaining -= int(chunk)
		i++
	}
	s.n = RATE_BYTES
}

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func (s *State) Squeeze256(out []byte) {
	s.squeeze(out, RATE_OF_CAP256_BYTES)
}

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func (s *State) Squeeze512(out []byte) {
	s.squeeze(out, RATE_OF_CAP512_BYTES)
}

// This function squeezes into XOR. pass a full-zero `out` if you want only extraction.
//
// empty `out` will do NOTHING -- state doesn't get permuted
func (s *State) Squeeze960(out []byte) {
	s.squeeze(out, RATE_OF_CAP960_BYTES)
}
