package internal

import "github.com/MHSarmadi/Twister"

type _Phase uint8

const (
	_PHASE_FRESH _Phase = iota
	_PHASE_READY
	_PHASE_ABSORBING
	_PHASE_SQUEEZING
)

type State struct {
	state     Twister.State1024
	streaming bool // decides whether squeeze should use cache/buffering or not
	phase     _Phase
	rate      int // holds absorption/squeeze rate.

	// The buffer is used for streaming mode.
	//
	// On Streaming Absorption (Phase: Absorbing), it would be an array buffer
	// (always start at zero), holding the pending input. once it hits the limit
	// (rate bytes), it absorbs it as a full block and drains out.
	//
	// On Streaming Squeezing (Phase: Squeezing), it would be an array buffer
	// (always start at zero), holding the pending output. once it hits the limit
	// (rate butes), drains out, permutes and gets ready for next
	// squeezing round.
	buffer [128]byte
	len    int // vaild bytes to read of buffer.

	domain uint64 // the domain separation. (Absorbing phase specific)
}

// Init resets the permutation state with the given 256-bit seed and moves
// to _PHASE_READY. Requires _PHASE_FRESH (zero value or after Wipe).
// Follow with BeginAbsorb or BeginSqueeze.
func (s *State) Init(initials [4]uint64) {
	if s.phase != _PHASE_FRESH {
		panic("Call on a fresh state. Wipe first.")
	}
	s.state = Twister.NewState1024_256bitSeed(initials)
	s.phase = _PHASE_READY
}

// Reset returns to _PHASE_READY so a new absorb/squeeze cycle can start.
// In _PHASE_ABSORBING (streaming) the pending message is sealed first —
// Reset completes the message rather than discarding it. In _PHASE_SQUEEZING
// the state is left as-is (post-absorb or post-last-permute).
func (s *State) Reset() {
	switch s.phase {
	case _PHASE_FRESH:
		panic("Init first.")
	case _PHASE_READY:
		return
	case _PHASE_ABSORBING:
		s.SealAbsorb()
	}
	s.streaming = false
	s.rate, s.len, s.domain = 0, 0, 0
	s.phase = _PHASE_READY
}

// Permute applies one permutation and clears any buffered output. In
// _PHASE_SQUEEZING (streaming) this discards pending bytes — next Squeeze
// refills from the advanced state. Panics in _PHASE_ABSORBING (SealAbsorb
// first) and _PHASE_FRESH.
func (s *State) Permute() {
	switch s.phase {
	case _PHASE_FRESH:
		panic("Init first.")
	case _PHASE_ABSORBING:
		panic("Can't permute mid-absorption; call SealAbsorb.")
	case _PHASE_READY, _PHASE_SQUEEZING:
		s.len = 0
		s.state.Permute()
	default:
		panic("Unknown phase.")
	}
}

// Clone returns a value copy; both copies evolve independently thereafter.
func (s *State) Clone() State {
	return *s
}

// Wipe zeroes the state, buffer, and bookkeeping, and moves to _PHASE_FRESH.
// Safe from any phase; the State is unusable until Init is called again.
func (s *State) Wipe() {
	s.state.Wipe()
	for i := range s.buffer {
		s.buffer[i] = 0
	}
	s.streaming = false
	s.rate, s.len, s.domain = 0, 0, 0
	s.phase = _PHASE_FRESH
}
