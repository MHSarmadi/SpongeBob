package internal

// CONSTANTS
const (
	RATE_OF_CAP256_BYTES = 96
	RATE_OF_CAP512_BYTES = 64
	RATE_OF_CAP960_BYTES = 8
)

// pad XORs the pad10*1 pattern and the rate-specific domain bits into
// rate_block. delta is the number of unfilled tail bytes (1 ≤ delta ≤
// len(rate_block)*8). The domain bits derive from the rate: 12 words → 1,
// 8 → 2, 1 → 3. Panics on delta <= 0 or unsupported rate.
func pad(rate_block []uint64, delta int) {
	if delta <= 0 {
		panic("invalid delta")
	}
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
