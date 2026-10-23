package internal

import "github.com/MHSarmadi/Twister"

type State struct {
	state Twister.State1024
	cache bool      // decides whether squeeze should use cache or not
	buf   [128]byte // ring buffer; must be >= max rate
	pos   int       // read position in buf (start of valid bytes)
	n     int       // valid bytes from pos
}

func (s *State) reset_cache(enable_cache bool) {
	s.cache = enable_cache
	s.pos = 0
	s.n = 0
}

func (s *State) Reset(enable_cache bool) {
	s.state = Twister.State1024{}
	s.reset_cache(enable_cache)
}

func (s *State) ResetFromExisting(src *Twister.State1024, enable_cache bool) {
	s.state = *src
	s.reset_cache(enable_cache)
}

func (s *State) Init(initials [4]uint64, enable_cache bool) {
	s.state = Twister.NewState1024_256bitSeed(initials)
	s.reset_cache(enable_cache)
}

// resetSqueezeCache drops any pending cached squeeze bytes. Called by
// absorb so a new absorb never leaks pre-absorb output to a later squeeze.
func (s *State) resetSqueezeCache() {
	s.pos = 0
	s.n = 0
}

func (s *State) Wipe() {
	s.state.Wipe()
	if s.cache {
		for i := range s.buf {
			s.buf[i] = 0
		}
	}
}
