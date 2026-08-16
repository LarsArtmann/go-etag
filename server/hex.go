package etag

// hexDigitsLower is the lowercase hex alphabet (0-9, a-f) used by the ETag
// encoder. Indexed by a 4-bit nibble to produce a single hex character byte.
const hexDigitsLower = "0123456789abcdef"

// hexEncodeUint64 returns the lowercase hex encoding of v as a 16-character
// string. The encoding is written into a stack-allocated array and converted
// to a string in a single allocation, avoiding the overhead of strings.Builder.
func hexEncodeUint64(v uint64) string {
	var buf [hashUint64HexChars]byte

	for i := hashUint64Bytes - 1; i >= 0; i-- {
		buf[i*hexCharsPerByte+1] = hexDigitsLower[v&hexNibbleMask]
		buf[i*hexCharsPerByte] = hexDigitsLower[(v>>hexNibbleShift)&hexNibbleMask]
		v >>= hexBitsPerByte
	}

	return string(buf[:])
}

const (
	hexNibbleShift  = 4
	hexNibbleMask   = 0x0f
	hexBitsPerByte  = 8
	hexCharsPerByte = 2
)
