package utils

import "testing"

func TestDedent(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
	}{
		{"trailing newline", "\n  first\n    second\n  ", "first\n  second\n"},
		{"no trailing newline", "\n\tfirst\n\tsecond", "first\nsecond"},
		{"no leading newline", "  first\n  second", "first\nsecond"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Dedent(tc.input); got != tc.want {
				t.Errorf("Dedent(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
