package main

import (
	"encoding/json"
	"strings"
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

func TestCanonicalDashboardContent_IgnoresUUIDsAndHiddenRow(t *testing.T) {
	withRow := &axonopsClient.CustomDashboard{Panels: []axonopsClient.CustomPanel{
		{UUID: "e", Type: rowPanelType, Title: emptyRowTitle, Layout: axonopsClient.PanelLayout{W: 18, H: 1}},
		{UUID: "a", Type: "counter", Title: "A", Group: "e", Details: json.RawMessage(`{"b":1, "a":2}`), Layout: axonopsClient.PanelLayout{Y: 1, W: 3, H: 2}},
	}}
	without := &axonopsClient.CustomDashboard{Panels: []axonopsClient.CustomPanel{
		{UUID: "z", Type: "counter", Title: "A", Details: json.RawMessage(`{"a":2,"b":1}`), Layout: axonopsClient.PanelLayout{W: 3, H: 2}},
	}}
	if canonicalDashboardContent(withRow) != canonicalDashboardContent(without) {
		t.Errorf("equivalent dashboards differ:\n%s\n%s", canonicalDashboardContent(withRow), canonicalDashboardContent(without))
	}
	without.Panels[0].Title = "B"
	if canonicalDashboardContent(withRow) == canonicalDashboardContent(without) {
		t.Error("different titles compared equal")
	}
}

func TestDashboardFromJSON_ReusesPanelUUIDsAndShiftsY(t *testing.T) {
	doc := `{"panels":[{"type":"counter","title":"A","layout":{"x":0,"y":0,"w":3,"h":2}}]}`
	existing := &axonopsClient.CustomDashboard{Panels: []axonopsClient.CustomPanel{
		{UUID: "row", Type: rowPanelType, Title: emptyRowTitle},
		{UUID: "old", Type: "counter", Title: "A"},
	}}
	got, err := dashboardFromJSON(doc, "id1", "N", existing)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "id1" || got.Name != "N" || len(got.Panels) != 2 {
		t.Fatalf("unexpected dashboard: %+v", got)
	}
	if got.Panels[0].UUID != "row" || got.Panels[1].UUID != "old" || got.Panels[1].Layout.I != "old" {
		t.Errorf("uuids not reused: %+v", got.Panels)
	}
	if got.Panels[1].Layout.Y != 1 || string(got.Panels[1].Details) != "{}" {
		t.Errorf("y=%d details=%s, want 1 and {}", got.Panels[1].Layout.Y, got.Panels[1].Details)
	}
}

func TestCanonicalDashboardContent_NilAndEmptyFiltersEqual(t *testing.T) {
	panels := []axonopsClient.CustomPanel{{Type: "counter", Title: "A", Layout: axonopsClient.PanelLayout{W: 1, H: 1}}}
	a := &axonopsClient.CustomDashboard{Panels: panels}
	b := &axonopsClient.CustomDashboard{Panels: panels, Filters: []axonopsClient.DashboardFilter{}}
	if canonicalDashboardContent(a) != canonicalDashboardContent(b) {
		t.Error("nil and empty filters compared different")
	}
}

func TestExportDashboardJSON_OmitsUUIDAndName(t *testing.T) {
	d := &axonopsClient.CustomDashboard{UUID: "u", Name: "n", Panels: []axonopsClient.CustomPanel{{Type: "counter", Title: "A", Layout: axonopsClient.PanelLayout{W: 1, H: 1}}}}
	got := exportDashboardJSON(d)
	if strings.Contains(got, `"name"`) || strings.Contains(got, `"uuid":"u"`) {
		t.Errorf("export keeps uuid or name: %s", got)
	}
	if _, err := parseDashboardJSON(got); err != nil {
		t.Errorf("export is not accepted as input: %v", err)
	}
}
