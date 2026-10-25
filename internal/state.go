package internal

import "github.com/MHSarmadi/Twister"

type State struct {
	state     Twister.State1024
	streaming bool // decides whether squeeze should use cache/buffering or not

	sq_buf [128]byte // ring buffer; must be >= max rate
	sq_pos int       // read position in buf (start of valid bytes)
	sq_n   int       // valid bytes from pos

	ab_buf    [128]byte // array buffer; must be >= max rate
	ab_len    int
	ab_ds     uint64 // domain separation
	ab_rate   int    // rate of the current absorb buffer
	ab_sealed bool
}

func (s *State) Init(initials [4]uint64, enable_cache bool) {
	s.state = Twister.NewState1024_256bitSeed(initials)
	s.ab_sealed = true
	s.reset_cache(enable_cache)
}

func (s *State) Permute() {
	s.state.Permute()
	s.reset_cache(s.streaming)
}

func (s *State) Clone() State {
	return *s
}

func (s *State) Wipe() {
	s.state.Wipe()
	for i := range s.sq_buf {
		s.sq_buf[i] = 0
	}
	for i := range s.ab_buf {
		s.ab_buf[i] = 0
	}
	s.sq_pos, s.sq_n, s.ab_len, s.ab_rate = 0, 0, 0, 0
	s.ab_sealed = true
}

func (s *State) reset_cache(enable_cache bool) {
	s.streaming = enable_cache
	s.sq_pos = 0
	s.sq_n = 0
}
