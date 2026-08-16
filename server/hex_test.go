package etag

import (
	"fmt"
	"testing"
)

func TestHexEncodeUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  uint64
		want string
	}{
		{name: "zero", val: 0, want: "0000000000000000"},
		{name: "one", val: 1, want: "0000000000000001"},
		{name: "max uint8", val: 0xff, want: "00000000000000ff"},
		{name: "max uint32", val: 0xffffffff, want: "00000000ffffffff"},
		{name: "max uint64", val: 0xffffffffffffffff, want: "ffffffffffffffff"},
		{name: "fnv-64a hello world", val: 0x779a65e7023cd2e7, want: "779a65e7023cd2e7"},
		{name: "fnv-64a empty", val: 0xcbf29ce484222325, want: "cbf29ce484222325"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := hexEncodeUint64(tt.val)
			if got != tt.want {
				t.Errorf("hexEncodeUint64(0x%x) = %q, want %q", tt.val, got, tt.want)
			}

			if len(got) != hashUint64HexChars {
				t.Errorf("len = %d, want %d", len(got), hashUint64HexChars)
			}
		})
	}
}

// TestHexEncodeUint64_MatchesFmtSprintf verifies our hand-rolled hex encoder
// produces the same output as the stdlib formatter across a range of values.
func TestHexEncodeUint64_MatchesFmtSprintf(t *testing.T) {
	t.Parallel()

	values := []uint64{
		0,
		1,
		0xa,
		0xff,
		0x100,
		0x1234,
		0xdeadbeef,
		0x7fffffffffffffff,
		0xffffffffffffffff,
		0x779a65e7023cd2e7,
	}

	for _, v := range values {
		expected := fmt.Sprintf("%016x", v)
		got := hexEncodeUint64(v)
		if got != expected {
			t.Errorf("hexEncodeUint64(0x%x) = %q, want %q (fmt.Sprintf)", v, got, expected)
		}
	}
}

func TestHexDigitsLower(t *testing.T) {
	t.Parallel()

	if hexDigitsLower != "0123456789abcdef" {
		t.Errorf("hexDigitsLower = %q, want %q", hexDigitsLower, "0123456789abcdef")
	}
}
