package bench

import (
	"fmt"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
	"github.com/MHSarmadi/Twister"
	"golang.org/x/crypto/sha3"
)

var zero_initials = [4]uint64{0, 0, 0, 0}

// ---------- shared setup ----------

// benchData returns a deterministic non-zero payload of n bytes. Deterministic
// (not crypto/rand) so runs are reproducible and setup time doesn't leak into
// the measurement.
func benchData(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*31 + 7)
	}
	return b
}

var (
	payloadSizes = []int{64, 1 << 10, 16 << 10, 1 << 20} // absorb / combined input
	squeezeSizes = []int{32, 1 << 10, 64 << 10}          // squeeze output
)

type spongeRate struct {
	name      string
	absorb    func(*internal.State, []byte, uint64)
	squeeze   func(*internal.State, []byte)
	rateBytes int
}

var spongeRates = []spongeRate{
	{"256", (*internal.State).Absorb256, (*internal.State).Squeeze256, internal.RATE_OF_CAP256_BYTES},
	{"512", (*internal.State).Absorb512, (*internal.State).Squeeze512, internal.RATE_OF_CAP512_BYTES},
	{"960", (*internal.State).Absorb960, (*internal.State).Squeeze960, internal.RATE_OF_CAP960_BYTES},
}

func cacheName(c bool) string {
	if c {
		return "cached"
	}
	return "uncached"
}

// ---------- SpongeBob: absorb ----------

func BenchmarkAbsorb(b *testing.B) {
	for _, r := range spongeRates {
		for _, size := range payloadSizes {
			data := benchData(size)
			b.Run(fmt.Sprintf("%s/%d", r.name, size), func(b *testing.B) {
				s := internal.State{}
				b.SetBytes(int64(size))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r.absorb(&s, data, 0xDEADBEEFCAFEBABE)
				}
			})
		}
	}
}

// ---------- SpongeBob: squeeze (cached + uncached) ----------

func BenchmarkSqueeze(b *testing.B) {
	for _, r := range spongeRates {
		for _, cache := range []bool{false, true} {
			for _, size := range squeezeSizes {
				name := fmt.Sprintf("%s/%d/%s", r.name, size, cacheName(cache))
				out := make([]byte, size)
				b.Run(name, func(b *testing.B) {
					s := internal.State{}
					s.Init(zero_initials, cache)
					r.absorb(&s, []byte("seed"), 0x1) // give the state something non-zero
					b.SetBytes(int64(size))
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						r.squeeze(&s, out)
					}
				})
			}
		}
	}
}

// ---------- SpongeBob: absorb + squeeze (one-shot, both cache modes) ----------

// one-shot absorb+squeeze, direct method calls so `s` stays on the stack
func benchOneShot(b *testing.B, rateBytes int, cache bool) {
	msg := benchData(1 << 10)
	out := make([]byte, 32)
	b.SetBytes(int64(len(msg)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s internal.State
		s.Init(zero_initials, cache)
		switch rateBytes {
		case internal.RATE_OF_CAP256_BYTES:
			s.Absorb256(msg, 0xDEADBEEFCAFEBABE)
			s.Squeeze256(out)
		case internal.RATE_OF_CAP512_BYTES:
			s.Absorb512(msg, 0xDEADBEEFCAFEBABE)
			s.Squeeze512(out)
		case internal.RATE_OF_CAP960_BYTES:
			s.Absorb960(msg, 0xDEADBEEFCAFEBABE)
			s.Squeeze960(out)
		default:
			b.Fatalf("bad rateBytes %d", rateBytes)
		}
	}
}

func BenchmarkAbsorbSqueeze(b *testing.B) {
	for _, r := range spongeRates {
		for _, cache := range []bool{false, true} {
			name := fmt.Sprintf("%s/%s", r.name, cacheName(cache))
			b.Run(name, func(b *testing.B) { benchOneShot(b, r.rateBytes, cache) })
		}
	}
}

// ---------- SpongeBob: bare permutation (the ceiling) ----------

func BenchmarkPermute(b *testing.B) {
	var st Twister.State1024
	b.ReportAllocs()

	for b.Loop() {
		st.Permute()
	}
}

// ---------- Reference: x/crypto/sha3 SHAKE (same sponge paradigm) ----------

type shakeAlg struct {
	name      string
	new       func() sha3.ShakeHash
	rateBytes int // for reference in the output table
}

func shakeAlgs() []shakeAlg {
	return []shakeAlg{
		{"SHAKE128", sha3.NewShake128, 168}, // capacity 256, rate 1344 bits
		{"SHAKE256", sha3.NewShake256, 136}, // capacity 512, rate 1088 bits
	}
}

func BenchmarkSHAKE_Absorb(b *testing.B) {
	for _, alg := range shakeAlgs() {
		for _, size := range payloadSizes {
			data := benchData(size)
			b.Run(fmt.Sprintf("%s/%d", alg.name, size), func(b *testing.B) {
				h := alg.new()
				b.SetBytes(int64(size))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					h.Write(data)
				}
			})
		}
	}
}

func BenchmarkSHAKE_Squeeze(b *testing.B) {
	for _, alg := range shakeAlgs() {
		for _, size := range squeezeSizes {
			b.Run(fmt.Sprintf("%s/%d", alg.name, size), func(b *testing.B) {
				h := alg.new()
				h.Write([]byte("seed"))
				out := make([]byte, size)
				b.SetBytes(int64(size))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					h.Read(out)
				}
			})
		}
	}
}

func BenchmarkSHAKE_AbsorbSqueeze(b *testing.B) {
	msg := benchData(1 << 10)
	out := make([]byte, 32)
	for _, alg := range shakeAlgs() {
		b.Run(alg.name, func(b *testing.B) {
			b.SetBytes(int64(len(msg)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h := alg.new()
				h.Write(msg)
				h.Read(out)
			}
		})
	}
}
