package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// parseDashboardJSON decodes a dashboard document in the shape the AxonOps
// API and UI export use and checks that it can be stored.
func parseDashboardJSON(raw string) (*axonopsClient.CustomDashboard, error) {
	var dash axonopsClient.CustomDashboard
	if err := json.Unmarshal([]byte(raw), &dash); err != nil {
		return nil, fmt.Errorf("not a dashboard JSON object: %w", err)
	}
	_, panels := splitEmptyRow(dash.Panels)
	if len(panels) == 0 {
		return nil, fmt.Errorf("dashboard has no panels")
	}
	for i, p := range panels {
		if p.Type == "" {
			return nil, fmt.Errorf("panel %d has no type", i)
		}
		if p.Layout.W < 1 || p.Layout.H < 1 || p.Layout.X < 0 || p.Layout.Y < 0 {
			return nil, fmt.Errorf("panel %d (%q) has an invalid layout", i, p.Title)
		}
	}
	for i, f := range dash.Filters {
		if f.Name == "" {
			return nil, fmt.Errorf("filter %d has no name", i)
		}
		if f.Type != "query" && f.Type != "custom" {
			return nil, fmt.Errorf("filter %q has type %q, want query or custom", f.Name, f.Type)
		}
	}
	return &dash, nil
}

// dashboardFromJSON builds the API dashboard from an imported JSON document.
// The dashboard UUID is id and the name is name: the document's own are
// ignored. Panel UUIDs are reused by position from existing so UI state tied
// to them survives an update, and are new otherwise, so a dashboard exported
// from another cluster never collides with an existing one. A hidden UI row in
// the document is kept; otherwise one is added and the panels move one grid
// row down below it.
func dashboardFromJSON(raw, id, name string, existing *axonopsClient.CustomDashboard) (axonopsClient.CustomDashboard, error) {
	parsed, err := parseDashboardJSON(raw)
	if err != nil {
		return axonopsClient.CustomDashboard{}, err
	}

	var oldPanels []axonopsClient.CustomPanel
	emptyRowUUID := ""
	if existing != nil {
		var row *axonopsClient.CustomPanel
		row, oldPanels = splitEmptyRow(existing.Panels)
		if row != nil {
			emptyRowUUID = row.UUID
		}
	}
	if emptyRowUUID == "" {
		emptyRowUUID = uuid.NewString()
	}

	docRow, panels := splitEmptyRow(parsed.Panels)
	yShift := 1
	if docRow != nil {
		yShift = 0
	}

	dash := axonopsClient.CustomDashboard{UUID: id, Name: name, Filters: parsed.Filters}
	dash.Panels = append(dash.Panels, axonopsClient.CustomPanel{
		UUID:    emptyRowUUID,
		Type:    rowPanelType,
		Title:   emptyRowTitle,
		Details: json.RawMessage(emptyRowDetails),
		Layout:  axonopsClient.PanelLayout{W: 18, H: 1, I: emptyRowUUID},
	})
	for i, p := range panels {
		panelUUID := uuid.NewString()
		if i < len(oldPanels) {
			panelUUID = oldPanels[i].UUID
		}
		p.UUID = panelUUID
		p.Group = ""
		p.Layout.I = panelUUID
		p.Layout.Y += yShift
		if len(p.Details) == 0 || string(p.Details) == "null" {
			p.Details = json.RawMessage(defaultPanelDetails(p.Type))
		}
		dash.Panels = append(dash.Panels, p)
	}
	return dash, nil
}

// canonicalDashboardContent renders the user-visible content of a dashboard
// (filters and panels, without UUIDs, grouping, the hidden UI row and its
// grid offset) in a form that is equal for equivalent dashboards. It is used
// to tell real drift from differences in formatting or generated UUIDs.
func canonicalDashboardContent(d *axonopsClient.CustomDashboard) string {
	type canonPanel struct {
		Title   string          `json:"title"`
		Type    string          `json:"type"`
		Details json.RawMessage `json:"details"`
		X       int             `json:"x"`
		Y       int             `json:"y"`
		W       int             `json:"w"`
		H       int             `json:"h"`
	}
	row, panels := splitEmptyRow(d.Panels)
	yOffset := 0
	if row != nil {
		yOffset = 1
	}
	out := struct {
		Filters []axonopsClient.DashboardFilter `json:"filters"`
		Panels  []canonPanel                    `json:"panels"`
	}{}
	if len(d.Filters) > 0 {
		out.Filters = d.Filters // nil and empty compare equal
	}
	for _, p := range panels {
		details := string(p.Details)
		if len(p.Details) == 0 || details == "null" {
			details = defaultPanelDetails(p.Type)
		}
		out.Panels = append(out.Panels, canonPanel{
			Title:   p.Title,
			Type:    p.Type,
			Details: json.RawMessage(normalizeJSON([]byte(details))),
			X:       p.Layout.X,
			Y:       max(p.Layout.Y-yOffset, 0),
			W:       p.Layout.W,
			H:       p.Layout.H,
		})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// exportDashboardJSON renders a dashboard as a JSON document that
// dashboardFromJSON accepts.
func exportDashboardJSON(d *axonopsClient.CustomDashboard) string {
	// The UUID and name are ignored on input, so leave them out.
	c := *d
	c.UUID, c.Name = "", ""
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return normalizeJSON(b)
}

// dashboardJSONValidator checks that a string is a dashboard document.
type dashboardJSONValidator struct{}

func (v dashboardJSONValidator) Description(_ context.Context) string {
	return "value must be a dashboard JSON document with at least one panel"
}

func (v dashboardJSONValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v dashboardJSONValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := parseDashboardJSON(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid dashboard JSON", strings.TrimSpace(err.Error()))
	}
}
