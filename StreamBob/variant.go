package streambob

import "errors"

type variant uint8

// Default: zero -> uninitialized -> panic!
const (
	Variant128 variant = iota + 1 // 256-bit output, rate 96
	Variant256                    // 512-bit output, rate 64
	Variant480                    // 960-bit output, rate 8
)

var errInvalidVariant = errors.New("StreamBob: Invalid Variant.")
var errUninitialized = errors.New("StreamBob: Uninitialized.")
