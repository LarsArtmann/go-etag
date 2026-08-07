package etag

// hexDigitsLower is the lowercase hex alphabet (0-9, a-f) used by the ETag
// encoder. Indexed by a 4-bit nibble to produce a single hex character byte.
const hexDigitsLower = "0123456789abcdef"
