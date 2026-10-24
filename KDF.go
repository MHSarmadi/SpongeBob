package spongebob

import (
	streambob "github.com/MHSarmadi/SpongeBob/StreamBob"
	"github.com/MHSarmadi/SpongeBob/internal"
)

func KDF(master, salt []byte, context string, output []byte) {
	var h streambob.Unkeyed
	defer h.Wipe()
	h.CustomInit(streambob.Variant128, context, internal.KDFUNC__)
	h.Absorb(salt)
	h.Permute()
	h.Absorb(master)
	h.Squeeze(output)
}
