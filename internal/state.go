package internal

import "github.com/MHSarmadi/Twister"

type State struct {
	state *Twister.State1024
	cache bool      // decides whether squeeze should use cache or not
	buf   [128]byte // ring buffer; must be >= max rate
	pos   int       // read position in buf (start of valid bytes)
	n     int       // valid bytes from pos
}

func NewState(state *Twister.State1024, enable_cache bool) *State {
	return &State{state: state, cache: enable_cache}
}

// resetSqueezeCache drops any pending cached squeeze bytes. Called by
// absorb so a new absorb never leaks pre-absorb output to a later squeeze.
func (s *State) resetSqueezeCache() {
	s.pos = 0
	s.n = 0
}
