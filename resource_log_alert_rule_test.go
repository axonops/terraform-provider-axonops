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
		{"leading minus token", "is -DOWN", "is -DOWN"},
		{"pipe operator between terms", "foo | bar", "foo | bar"},
		{"double pipe operator", "foo || bar", "foo || bar"},
		{"quoted phrase", `"is now DOWN"`, `"is now DOWN"`},
		{"parens get normalised", "Unable to lock JVM memory (ENOMEM)",
			"+Unable +to +lock +JVM +memory +(ENOMEM)"},
		{"wildcard token gets normalised", "error foo*", "+error +foo*"},
		{"tilde token gets normalised", "foo~2 bar", "+foo~2 +bar"},
		{"single word with wildcard", "foo*", "foo*"},
		{"single word with parens", "(foo)", "(foo)"},
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
