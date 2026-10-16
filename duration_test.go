package main

import "testing"

func TestDurationsEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"24h", "1d", true},
		{"24h", "1440m", true},
		{"1d", "1440m", true},
		{"1h30m", "90m", true},
		{"1w", "7d", true},
		{"1m", "1m", true},
		{"24h", "25h", false},
		{"1m", "60", false},
		{"bogus", "bogus", true},
		{"bogus", "1m", false},
		{"300y", "1m", false},
		{"9999999999999999w", "9999999999999999w1ms", false},
	}
	for _, c := range cases {
		if got := durationsEqual(c.a, c.b); got != c.want {
			t.Errorf("durationsEqual(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestPreferPriorDuration(t *testing.T) {
	if got := preferPriorDuration("24h", "1d"); got != "24h" {
		t.Errorf("equivalent: got %q, want prior 24h", got)
	}
	if got := preferPriorDuration("24h", "2d"); got != "2d" {
		t.Errorf("drift: got %q, want API value 2d", got)
	}
	if got := preferPriorDuration("", "1d"); got != "1d" {
		t.Errorf("no prior: got %q, want 1d", got)
	}
}
