package spongebob

import streambob "github.com/MHSarmadi/SpongeBob/StreamBob"

func MAC(secret, data []byte, context string, output []byte) {
	var h streambob.Keyed
	defer h.Wipe()
	h.Init128(secret, context)
	h.Absorb(data)
	h.Squeeze(output)
}
