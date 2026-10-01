package main

import "testing"

func TestAlertRouteID(t *testing.T) {
	got := alertRouteID("cassandra", "my-cluster", "global", "error", "pagerduty", "ops-pagerduty")
	want := "cassandra/my-cluster/global/error/pagerduty/ops-pagerduty"
	if got != want {
		t.Errorf("alertRouteID() = %q, want %q", got, want)
	}
}

func TestValidRouteTypesMatchesRouteTypeMap(t *testing.T) {
	if len(validRouteTypes) != len(routeTypeMap) {
		t.Fatalf("validRouteTypes has %d entries, routeTypeMap has %d", len(validRouteTypes), len(routeTypeMap))
	}
	for _, rt := range validRouteTypes {
		if _, ok := routeTypeMap[rt]; !ok {
			t.Errorf("validRouteTypes contains %q which is not a key of routeTypeMap", rt)
		}
	}
}
