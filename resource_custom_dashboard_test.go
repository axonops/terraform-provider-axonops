package main

import (
	"encoding/json"
	"testing"

	axonopsClient "terraform-provider-axonops/client"
)

func TestNormalizeJSON(t *testing.T) {
	tests := []struct{ in, want string }{
		{`{"b": 1, "a": [ {"y":2,"x":1} ]}`, `{"a":[{"x":1,"y":2}],"b":1}`},
		{`{"max": 1000000, "f": 0.5}`, `{"f":0.5,"max":1000000}`},
		// Terraform's jsonencode escapes <, > and & the same way.
		{`{"q": "a<b"}`, `{"q":"a` + `\` + `u003cb"}`},
		{`not json`, `not json`},
	}
	for _, tt := range tests {
		if got := normalizeJSON([]byte(tt.in)); got != tt.want {
			t.Errorf("normalizeJSON(%s) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestDefaultPanelDetails_ValidJSON(t *testing.T) {
	for _, typ := range []string{rowPanelType, "line-chart"} {
		if !json.Valid([]byte(defaultPanelDetails(typ))) {
			t.Errorf("defaultPanelDetails(%q) is not valid JSON", typ)
		}
	}
}

func TestSplitEmptyRow(t *testing.T) {
	empty := axonopsClient.CustomPanel{UUID: "e", Type: rowPanelType, Title: emptyRowTitle}
	a := axonopsClient.CustomPanel{UUID: "a", Type: rowPanelType, Title: "A"}
	b := axonopsClient.CustomPanel{UUID: "b", Type: "line-chart", Title: emptyRowTitle}

	tests := []struct {
		name     string
		in       []axonopsClient.CustomPanel
		wantRow  string
		wantRest []string
	}{
		{"first", []axonopsClient.CustomPanel{empty, a, b}, "e", []string{"a", "b"}},
		{"not first", []axonopsClient.CustomPanel{a, empty, b}, "e", []string{"a", "b"}},
		{"absent", []axonopsClient.CustomPanel{a, b}, "", []string{"a", "b"}},
		{"nil", nil, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, rest := splitEmptyRow(tt.in)
			gotRow := ""
			if row != nil {
				gotRow = row.UUID
			}
			var gotRest []string
			for _, p := range rest {
				gotRest = append(gotRest, p.UUID)
			}
			if gotRow != tt.wantRow || len(gotRest) != len(tt.wantRest) {
				t.Fatalf("got row %q rest %v, want %q %v", gotRow, gotRest, tt.wantRow, tt.wantRest)
			}
			for i := range gotRest {
				if gotRest[i] != tt.wantRest[i] {
					t.Fatalf("got rest %v, want %v", gotRest, tt.wantRest)
				}
			}
		})
	}
}
