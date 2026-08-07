package etag

import "strings"

// hexDigitsLower is the lowercase hex alphabet (0-9, a-f) used by the ETag
// encoder. Indexed by a 4-bit nibble to produce a single hex character byte.
const hexDigitsLower = "0123456789abcdef"

// hexEncode returns the lowercase hex encoding of src.
// Each byte produces two hex characters.
func hexEncode(src []byte) string {
	var builder strings.Builder

	builder.Grow(len(src) * hexCharsPerByte)

	for _, b := range src {
		builder.WriteByte(hexDigitsLower[b>>hexNibbleShift])
		builder.WriteByte(hexDigitsLower[b&hexNibbleMask])
	}

	return builder.String()
}

const (
	hexNibbleShift  = 4
	hexCharsPerByte = 2
	hexNibbleMask   = 0x0f
)
