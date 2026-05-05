package main

import "testing"

func TestNormaliseLogContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single word", "DOWN", "DOWN"},
		{"multi word", "is now DOWN", "+is +now +DOWN"},
		{"already plus prefixed", "+is +now +DOWN", "+is +now +DOWN"},
		{"contains minus operator", "is -DOWN", "is -DOWN"},
		{"contains pipe operator", "foo | bar", "foo | bar"},
		{"contains wildcard", "foo*", "foo*"},
		{"contains parens", "(foo bar)", "(foo bar)"},
		{"contains tilde", "foo~2", "foo~2"},
		{"quoted phrase", `"is now DOWN"`, `"is now DOWN"`},
		{"leading and trailing whitespace", "  is now DOWN  ", "+is +now +DOWN"},
		{"tabs collapsed", "is\tnow\tDOWN", "+is +now +DOWN"},
		{"two words", "node down", "+node +down"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normaliseLogContent(tc.in)
			if got != tc.want {
				t.Errorf("normaliseLogContent(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
