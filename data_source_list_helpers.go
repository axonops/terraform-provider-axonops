package main

import (
	"fmt"
	"sort"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// configureDataSourceClient extracts the AxonOps HTTP client from provider
// data. It returns nil (without error) when the provider is not configured
// yet, which the framework does during validation.
func configureDataSourceClient(req datasource.ConfigureRequest, diags *diag.Diagnostics) *axonopsClient.AxonopsHttpClient {
	if req.ProviderData == nil {
		return nil
	}
	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		diags.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return nil
	}
	return client
}

// matchesFilter reports whether value passes an optional string filter. A
// null or unknown filter matches everything.
func matchesFilter(filter types.String, value string) bool {
	if filter.IsNull() || filter.IsUnknown() {
		return true
	}
	return filter.ValueString() == value
}

// sortedKeys returns the keys of m in lexical order so list data sources
// produce stable output regardless of API or map ordering.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// stringListValue converts a Go string slice into a Terraform list value,
// returning an empty (not null) list for a nil slice.
func stringListValue(values []string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}
