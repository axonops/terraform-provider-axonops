package main

import (
	"strings"
	"testing"
	"testing/quick"
)

func TestDeterministicAlertRuleID_Deterministic(t *testing.T) {
	a := deterministicAlertRuleID("org1", "cassandra", "prod", "HighGCPause", "metric")
	b := deterministicAlertRuleID("org1", "cassandra", "prod", "HighGCPause", "metric")
	if a != b {
		t.Fatalf("want identical UUIDs for identical inputs, got %q and %q", a, b)
	}
}

func TestDeterministicAlertRuleID_KnownValue(t *testing.T) {
	// Pin the output for a fixed set of inputs to guard against silent
	// hash-function regressions.
	const want = "e150cafb-6e26-5ef1-8e96-e6822ba634a1"
	got := deterministicAlertRuleID("org1", "cassandra", "prod", "HighGCPause", "metric")
	if got != want {
		t.Fatalf("known-value regression: want %q, got %q", want, got)
	}
}

func TestDeterministicAlertRuleID_UniquenessTable(t *testing.T) {
	base := []string{"org1", "cassandra", "prod", "HighGCPause", "metric"}
	mutations := []struct {
		field int
		value string
	}{
		{0, "org2"},
		{1, "kafka"},
		{2, "staging"},
		{3, "HighDiskUsage"},
		{4, "log"},
	}

	baseID := deterministicAlertRuleID(base[0], base[1], base[2], base[3], base[4])
	seen := map[string]string{baseID: "base"}
	for _, m := range mutations {
		args := append([]string(nil), base...)
		args[m.field] = m.value
		got := deterministicAlertRuleID(args[0], args[1], args[2], args[3], args[4])
		if prev, ok := seen[got]; ok {
			t.Errorf("collision: mutation field=%d value=%q produced same UUID as %s", m.field, m.value, prev)
		}
		seen[got] = m.value
	}
}

func TestDeterministicAlertRuleID_EmptyFields_NoPanic(t *testing.T) {
	got := deterministicAlertRuleID("", "", "", "", "")
	if got == "" {
		t.Fatal("expected non-empty UUID for empty inputs")
	}
	if len(got) != 36 || strings.Count(got, "-") != 4 {
		t.Fatalf("expected 36-char UUID with 4 dashes, got %q", got)
	}
}

func TestDeterministicAlertRuleID_ArbitraryInputs_NoPanic(t *testing.T) {
	prop := func(a, b, c, d, e string) bool {
		got := deterministicAlertRuleID(a, b, c, d, e)
		return len(got) == 36 && strings.Count(got, "-") == 4
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Fatal(err)
	}
}
