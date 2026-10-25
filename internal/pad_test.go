package internal

import "testing"

// expectedPad returns the block that pad(block, delta) should produce when
// starting from an all-zero block, for the given rate/domain.
func expectedPad(rateWords, delta int, domain uint64) []uint64 {
	block := make([]uint64, rateWords)
	dfw := (delta - 1) / 8
	rem := ((delta-1)%8 + 1) * 8
	boundary := rateWords - dfw - 1
	block[boundary] |= 1 << (rem - 1)
	block[rateWords-1] |= domain | 4
	return block
}

func TestPad_BitPlacement(t *testing.T) {
	cases := []struct {
		name      string
		rateWords int
		rateBytes int
		domain    uint64
	}{
		{"256", 12, 96, 1},
		{"512", 8, 64, 2},
		{"960", 1, 8, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for delta := 1; delta <= c.rateBytes; delta++ {
				block := make([]uint64, c.rateWords)
				pad(block, delta)
				want := expectedPad(c.rateWords, delta, c.domain)
				for i := range block {
					if block[i] != want[i] {
						t.Errorf("delta=%d word %d: got %#016x want %#016x",
							delta, i, block[i], want[i])
					}
				}
			}
		})
	}
}

func TestPad_Injective(t *testing.T) {
	cases := []struct {
		name      string
		rateWords int
		rateBytes int
	}{
		{"256", 12, 96},
		{"512", 8, 64},
		{"960", 1, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			seen := make(map[string]int)
			for delta := 1; delta <= c.rateBytes; delta++ {
				block := make([]uint64, c.rateWords)
				pad(block, delta)
				var key []byte
				for _, w := range block {
					key = append(key, byte(w>>56), byte(w>>48), byte(w>>40), byte(w>>32),
						byte(w>>24), byte(w>>16), byte(w>>8), byte(w))
				}
				if prev, ok := seen[string(key)]; ok {
					t.Errorf("delta=%d collides with delta=%d", delta, prev)
				}
				seen[string(key)] = delta
			}
		})
	}
}

func TestPad_PreservesDataBits(t *testing.T) {
	// Fill the block with 0xFF, then pad. Bytes above the pad region (i.e.
	// the first rateBytes-delta bytes) must be unchanged.
	cases := []struct {
		name      string
		rateWords int
		rateBytes int
	}{
		{"256", 12, 96},
		{"512", 8, 64},
		{"960", 1, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for delta := 1; delta <= c.rateBytes; delta++ {
				block := make([]uint64, c.rateWords)
				for i := range block {
					block[i] = ^uint64(0)
				}
				original := append([]uint64{}, block...)
				pad(block, delta)

				dataBytes := c.rateBytes - delta
				fullDataWords := dataBytes / 8
				remDataBytes := dataBytes % 8

				for w := 0; w < fullDataWords; w++ {
					if block[w] != original[w] {
						t.Errorf("delta=%d word %d (data) changed", delta, w)
					}
				}
				if remDataBytes > 0 {
					w := fullDataWords
					mask := ^uint64(0) << uint((8-remDataBytes)*8)
					if block[w]&mask != original[w]&mask {
						t.Errorf("delta=%d word %d (partial data) top bits changed", delta, w)
					}
				}
			}
		})
	}
}

func TestPad_UnsupportedRatePanics(t *testing.T) {
	for _, rateWords := range []int{0, 2, 4, 16} {
		t.Run("", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("pad on rateWords=%d did not panic", rateWords)
				}
			}()
			pad(make([]uint64, rateWords), 1)
		})
	}
}
