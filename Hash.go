package spongebob

import streambob "github.com/MHSarmadi/SpongeBob/StreamBob"

func Hash128(data []byte, context string, output []byte) {
	var h streambob.Unkeyed
	defer h.Wipe()
	h.Init128(context)
	h.Write(data)
	h.Read(output)
}

func Hash256(data []byte, context string, output []byte) {
	var h streambob.Unkeyed
	defer h.Wipe()
	h.Init256(context)
	h.Write(data)
	h.Read(output)
}

func Hash480(data []byte, context string, output []byte) {
	var h streambob.Unkeyed
	defer h.Wipe()
	h.Init480(context)
	h.Write(data)
	h.Read(output)
}
