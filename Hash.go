package spongebob

import (
	"github.com/MHSarmadi/SpongeBob/internal"
)

func Hash512(data []byte, context string, output []byte) {
	var state internal.State
	defer state.Wipe()

	state.Init([4]uint64{_SP0NGB0B, _PR1M1T1V, _SARMADI_, __HASH___}, false)

	state.Absorb256([]byte(context), _C0NT3XT_)
	state.Absorb512(data, _PLA1NTXT)

	state.Squeeze512(output)
}

func Hash960(data []byte, context string, output []byte) {
	var state internal.State
	defer state.Wipe()

	state.Init([4]uint64{_SP0NGB0B, _PR1M1T1V, _SARMADI_, __HASH___}, false)

	state.Absorb256([]byte(context), _C0NT3XT_)
	state.Absorb960(data, _PLA1NTXT)

	state.Squeeze960(output)
}
