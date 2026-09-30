package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseConnectorImportID_Simple_Succeeds(t *testing.T) {
	cluster, connectCluster, name, err := parseConnectorImportID("mycluster/connect1/my-connector")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cluster != "mycluster" || connectCluster != "connect1" || name != "my-connector" {
		t.Fatalf("unexpected parse result: %q %q %q", cluster, connectCluster, name)
	}
}

func TestParseConnectorImportID_NameWithSlash_AbsorbsRemainder(t *testing.T) {
	cluster, connectCluster, name, err := parseConnectorImportID("mycluster/connect1/team/my-connector")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cluster != "mycluster" || connectCluster != "connect1" || name != "team/my-connector" {
		t.Fatalf("unexpected parse result: %q %q %q", cluster, connectCluster, name)
	}
}

func TestParseConnectorImportID_TooFewFields_ReturnsError(t *testing.T) {
	_, _, _, err := parseConnectorImportID("mycluster/connect1")
	if err == nil {
		t.Fatal("expected error for import ID with too few fields")
	}
}

func TestRefreshConnectorConfig_DropsMissingKeys(t *testing.T) {
	managed := map[string]types.String{
		"topics":      types.StringValue("orders"),
		"tasks.max":   types.StringValue("1"),
		"was.removed": types.StringValue("x"),
	}
	remote := map[string]string{
		"topics":          "orders-updated",
		"tasks.max":       "1",
		"name":            "my-connector",
		"server.injected": "abc",
	}

	got := refreshConnectorConfig(managed, remote)

	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(got), got)
	}
	if got["topics"].ValueString() != "orders-updated" {
		t.Fatalf("expected topics to refresh to %q, got %q", "orders-updated", got["topics"].ValueString())
	}
	if got["tasks.max"].ValueString() != "1" {
		t.Fatalf("expected tasks.max to stay %q, got %q", "1", got["tasks.max"].ValueString())
	}
	if _, ok := got["was.removed"]; ok {
		t.Fatal("expected was.removed to be dropped since missing server-side")
	}
	if _, ok := got["server.injected"]; ok {
		t.Fatal("expected server.injected to not be added since not managed")
	}
	if _, ok := got["name"]; ok {
		t.Fatal("expected name to never appear in refreshed config")
	}
}

func TestRefreshConnectorConfig_NilManaged_ReturnsNil(t *testing.T) {
	got := refreshConnectorConfig(nil, map[string]string{"a": "b"})
	if got != nil {
		t.Fatalf("expected nil result for nil managed map, got %v", got)
	}
}
