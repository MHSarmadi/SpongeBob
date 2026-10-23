package spongebob

import "github.com/MHSarmadi/SpongeBob/internal"

func MAC(secret, data []byte, context string, output []byte) {
	var state internal.State
	defer state.Wipe()

	state.Init([4]uint64{_SP0NGB0B, _PR1M1T1V, _SARMADI_, __MAC____}, false)

	state.Absorb256([]byte(context), _C0NT3XT_)
	state.Absorb960(secret, _S3CR3T__)
	state.Absorb512(data, _PLA1NTXT)

	state.Squeeze512(output)
}
