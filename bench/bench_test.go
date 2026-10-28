package bench

import (
	"fmt"
	"testing"

	"github.com/MHSarmadi/SpongeBob/internal"
	"github.com/MHSarmadi/Twister"
	"golang.org/x/crypto/sha3"
)

var zero_initials = [4]uint64{0, 0, 0, 0}

func benchData(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*31 + 7)
	}
	return b
}

var (
	payloadSizes = []int{64, 1 << 10, 16 << 10, 1 << 20}
	squeezeSizes = []int{32, 1 << 10, 64 << 10}
)

type spongeRate struct {
	name      string
	rateBytes int
}

var spongeRates = []spongeRate{
	{"256", internal.RATE_OF_CAP256_BYTES},
	{"512", internal.RATE_OF_CAP512_BYTES},
	{"960", internal.RATE_OF_CAP960_BYTES},
}

func modeName(stream bool) string {
	if stream {
		return "streaming"
	}
	return "non-streaming"
}

// ---------- SpongeBob: absorb ----------

func BenchmarkAbsorb(b *testing.B) {
	for _, r := range spongeRates {
		for _, size := range payloadSizes {
			data := benchData(size)
			b.Run(fmt.Sprintf("%s/%d", r.name, size), func(b *testing.B) {
				var s internal.State
				s.Init(zero_initials)
				b.SetBytes(int64(size))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					s.BeginAbsorb(r.rateBytes, false, 0xDEADBEEFCAFEBABE)
					s.Absorb(data)
				}
			})
		}
	}
}

// ---------- SpongeBob: squeeze (streaming + non-streaming) ----------

func BenchmarkSqueeze(b *testing.B) {
	for _, r := range spongeRates {
		for _, stream := range []bool{false, true} {
			for _, size := range squeezeSizes {
				name := fmt.Sprintf("%s/%d/%s", r.name, size, modeName(stream))
				b.Run(name, func(b *testing.B) {
					out := make([]byte, size)
					var s internal.State
					s.Init(zero_initials)
					s.BeginAbsorb(r.rateBytes, false, 0)
					s.Absorb([]byte("seed"))
					s.BeginSqueeze(r.rateBytes, stream)
					b.SetBytes(int64(size))
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						s.Squeeze(out)
					}
				})
			}
		}
	}
}

// ---------- SpongeBob: one-shot absorb + squeeze ----------

func benchOneShot(b *testing.B, rateBytes int, stream bool) {
	msg := benchData(1 << 10)
	out := make([]byte, 32)
	b.SetBytes(int64(len(msg)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s internal.State
		s.Init(zero_initials)
		s.BeginAbsorb(rateBytes, stream, 0xDEADBEEFCAFEBABE)
		s.Absorb(msg)
		s.SealAbsorb() // no-op if non-streaming already sealed
		s.BeginSqueeze(rateBytes, stream)
		s.Squeeze(out)
	}
}

func BenchmarkAbsorbSqueeze(b *testing.B) {
	for _, r := range spongeRates {
		for _, stream := range []bool{false, true} {
			name := fmt.Sprintf("%s/%s", r.name, modeName(stream))
			b.Run(name, func(b *testing.B) { benchOneShot(b, r.rateBytes, stream) })
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

// ---------- Reference: x/crypto/sha3 SHAKE ----------

type shakeAlg struct {
	name string
	new  func() sha3.ShakeHash
}

func shakeAlgs() []shakeAlg {
	return []shakeAlg{
		{"SHAKE128", sha3.NewShake128},
		{"SHAKE256", sha3.NewShake256},
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
