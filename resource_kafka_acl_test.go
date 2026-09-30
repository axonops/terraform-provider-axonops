package main

import (
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func aclData(resourceType, resourceName, patternType, principal, host, operation, permission string) aclResourceData {
	return aclResourceData{
		ResourceType:        types.StringValue(resourceType),
		ResourceName:        types.StringValue(resourceName),
		ResourcePatternType: types.StringValue(patternType),
		Principal:           types.StringValue(principal),
		Host:                types.StringValue(host),
		Operation:           types.StringValue(operation),
		PermissionType:      types.StringValue(permission),
	}
}

func TestFindACL_MatchFound_ReturnsTrue(t *testing.T) {
	data := aclData("TOPIC", "orders", "LITERAL", "User:alice", "*", "READ", "ALLOW")

	resp := &axonopsClient.ACLResponse{
		ACLResources: []axonopsClient.ACLResource{
			{
				ResourceType:        "TOPIC",
				ResourceName:        "orders",
				ResourcePatternType: "LITERAL",
				ACLs: []axonopsClient.KafkaACL{
					{Principal: "User:alice", Host: "*", Operation: "READ", PermissionType: "ALLOW"},
				},
			},
		},
	}

	if !findACL(data, resp) {
		t.Fatal("expected findACL to find a matching entry")
	}
}

func TestFindACL_CaseInsensitiveEnums_ReturnsTrue(t *testing.T) {
	data := aclData("TOPIC", "orders", "LITERAL", "User:alice", "*", "READ", "ALLOW")

	resp := &axonopsClient.ACLResponse{
		ACLResources: []axonopsClient.ACLResource{
			{
				ResourceType:        "topic",
				ResourceName:        "orders",
				ResourcePatternType: "literal",
				ACLs: []axonopsClient.KafkaACL{
					{Principal: "User:alice", Host: "*", Operation: "read", PermissionType: "allow"},
				},
			},
		},
	}

	if !findACL(data, resp) {
		t.Fatal("expected findACL to match case-insensitively on enum fields")
	}
}

func TestFindACL_NoMatch_ReturnsFalse(t *testing.T) {
	data := aclData("TOPIC", "orders", "LITERAL", "User:alice", "*", "READ", "ALLOW")

	resp := &axonopsClient.ACLResponse{
		ACLResources: []axonopsClient.ACLResource{
			{
				ResourceType:        "TOPIC",
				ResourceName:        "other-topic",
				ResourcePatternType: "LITERAL",
				ACLs: []axonopsClient.KafkaACL{
					{Principal: "User:alice", Host: "*", Operation: "READ", PermissionType: "ALLOW"},
				},
			},
		},
	}

	if findACL(data, resp) {
		t.Fatal("expected findACL to not match a different resource_name")
	}
}

func TestFindACL_NilResponse_ReturnsFalse(t *testing.T) {
	data := aclData("TOPIC", "orders", "LITERAL", "User:alice", "*", "READ", "ALLOW")

	if findACL(data, nil) {
		t.Fatal("expected findACL to return false for nil response")
	}
}

func TestFindACL_EmptyResources_ReturnsFalse(t *testing.T) {
	data := aclData("TOPIC", "orders", "LITERAL", "User:alice", "*", "READ", "ALLOW")

	if findACL(data, &axonopsClient.ACLResponse{}) {
		t.Fatal("expected findACL to return false for empty ACLResources")
	}
}

func TestParseACLImportID_SimplePrincipal_Succeeds(t *testing.T) {
	id := "mycluster/TOPIC/orders/LITERAL/User:alice/*/READ/ALLOW"

	cluster, resType, resName, patternType, principal, host, op, perm, err := parseACLImportID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cluster != "mycluster" || resType != "TOPIC" || resName != "orders" || patternType != "LITERAL" ||
		principal != "User:alice" || host != "*" || op != "READ" || perm != "ALLOW" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q %q %q %q", cluster, resType, resName, patternType, principal, host, op, perm)
	}
}

func TestParseACLImportID_PrincipalWithSlash_AbsorbsExtraSegments(t *testing.T) {
	id := "mycluster/TOPIC/orders/LITERAL/User:svc/account/*/READ/ALLOW"

	_, _, _, _, principal, host, op, perm, err := parseACLImportID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if principal != "User:svc/account" {
		t.Fatalf("expected principal %q, got %q", "User:svc/account", principal)
	}
	if host != "*" || op != "READ" || perm != "ALLOW" {
		t.Fatalf("unexpected trailing fields: host=%q op=%q perm=%q", host, op, perm)
	}
}

func TestParseACLImportID_TooFewFields_ReturnsError(t *testing.T) {
	_, _, _, _, _, _, _, _, err := parseACLImportID("mycluster/TOPIC/orders")
	if err == nil {
		t.Fatal("expected error for import ID with too few fields")
	}
}
