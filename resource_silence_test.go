package main

import (
	"testing"

	axonopsClient "terraform-provider-axonops/client"
)

func TestSilenceEqual(t *testing.T) {
	base := axonopsClient.SilenceWindow{
		Active:      true,
		CronExpr:    "0 * * * *",
		IsRecurring: false,
		Duration:    "1h",
		DCs:         []string{"dc1", "dc2"},
	}

	cases := []struct {
		name string
		a, b axonopsClient.SilenceWindow
		want bool
	}{
		{"identical", base, base, true},
		{"different ID is ignored", withID(base, "a"), withID(base, "b"), true},
		{"different active", base, withActive(base, false), false},
		{"different cron", base, withCron(base, "0 0 * * *"), false},
		{"different duration", base, withDuration(base, "2h"), false},
		{"different dc count", base, withDCs(base, []string{"dc1"}), false},
		{"different dc order", base, withDCs(base, []string{"dc2", "dc1"}), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := silenceEqual(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("silenceEqual() = %v, want %v", got, tc.want)
			}
		})
	}
}

func withID(s axonopsClient.SilenceWindow, id string) axonopsClient.SilenceWindow {
	s.ID = id
	return s
}

func withActive(s axonopsClient.SilenceWindow, active bool) axonopsClient.SilenceWindow {
	s.Active = active
	return s
}

func withCron(s axonopsClient.SilenceWindow, cron string) axonopsClient.SilenceWindow {
	s.CronExpr = cron
	return s
}

func withDuration(s axonopsClient.SilenceWindow, d string) axonopsClient.SilenceWindow {
	s.Duration = d
	return s
}

func withDCs(s axonopsClient.SilenceWindow, dcs []string) axonopsClient.SilenceWindow {
	s.DCs = dcs
	return s
}

func TestFindNewSilenceID(t *testing.T) {
	want := axonopsClient.SilenceWindow{Active: true, CronExpr: "0 * * * *", Duration: "1h", DCs: []string{}}

	t.Run("server honours generated ID", func(t *testing.T) {
		before := []axonopsClient.SilenceWindow{{ID: "existing"}}
		after := []axonopsClient.SilenceWindow{{ID: "existing"}, withID(want, "generated")}
		got := findNewSilenceID(before, after, "generated", want)
		if got != "generated" {
			t.Errorf("got %q, want %q", got, "generated")
		}
	})

	t.Run("server assigns its own ID, single new entry", func(t *testing.T) {
		before := []axonopsClient.SilenceWindow{{ID: "existing"}}
		after := []axonopsClient.SilenceWindow{{ID: "existing"}, withID(want, "server-assigned")}
		got := findNewSilenceID(before, after, "generated-but-unused", want)
		if got != "server-assigned" {
			t.Errorf("got %q, want %q", got, "server-assigned")
		}
	})

	t.Run("multiple new entries, match on fields", func(t *testing.T) {
		other := withDuration(want, "30m")
		before := []axonopsClient.SilenceWindow{}
		after := []axonopsClient.SilenceWindow{withID(other, "other-new"), withID(want, "matching-new")}
		got := findNewSilenceID(before, after, "generated-but-unused", want)
		if got != "matching-new" {
			t.Errorf("got %q, want %q", got, "matching-new")
		}
	})

	t.Run("no new entries falls back to generated ID", func(t *testing.T) {
		before := []axonopsClient.SilenceWindow{{ID: "existing"}}
		after := []axonopsClient.SilenceWindow{{ID: "existing"}}
		got := findNewSilenceID(before, after, "generated", want)
		if got != "generated" {
			t.Errorf("got %q, want %q", got, "generated")
		}
	})
}
