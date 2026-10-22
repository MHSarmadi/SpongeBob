package internal

// CONSTANTS
const (
	RATE_OF_CAP256_BYTES = 96
	RATE_OF_CAP512_BYTES = 64
	RATE_OF_CAP960_BYTES = 8
)

// 10*1dd PAD (dd is domain bits)
func pad(rate_block []uint64, delta int) {
	rate := len(rate_block)

	var domain uint64
	switch rate {
	// 256
	case 12:
		domain = 1
	// 512
	case 8:
		domain = 2
	// 960
	case 1:
		domain = 3
	default:
		panic("pad: unsupported rate")
	}

	delta_full_words := (delta - 1) / 8 // ceil(delta / 8) - 1

	delta_pad_region_bits := (uint64(delta-1)%8 + 1) * 8 // 8 * (delta % 8) where 0 -> 64 == 1 full word

	rate_block[rate-delta_full_words-1] ^= 1 << (delta_pad_region_bits - 1)
	rate_block[rate-1] ^= domain | 4
}
