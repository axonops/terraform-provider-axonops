package axonopsClient

import (
	"context"
	"encoding/json"
	"fmt"
)

// dashboardTemplateVersion is the dashboard template format edited by the
// AxonOps UI. Without it the server reads and writes the legacy 1.0 template.
const dashboardTemplateVersion = "2.0"

// DashboardTemplate is the full set of dashboards stored for one cluster.
// The server stores it as a single document: every PUT replaces all
// dashboards of the cluster, so callers must read-modify-write.
type DashboardTemplate struct {
	Type       string            `json:"type,omitempty"`
	Dashboards []CustomDashboard `json:"dashboards,omitempty"`
}

type CustomDashboard struct {
	UUID    string            `json:"uuid,omitempty"`
	Name    string            `json:"name,omitempty"`
	Filters []DashboardFilter `json:"filters,omitempty"`
	Panels  []CustomPanel     `json:"panels,omitempty"`
}

type DashboardFilter struct {
	Name        string  `json:"name,omitempty"`
	Label       string  `json:"label,omitempty"`
	Type        string  `json:"type,omitempty"`
	Multi       *bool   `json:"multi,omitempty"`
	CustomLogic string  `json:"customLogic,omitempty"`
	Limit       int     `json:"limit,omitempty"`
	Query       string  `json:"query,omitempty"`
	Regex       string  `json:"regex,omitempty"`
	Values      *string `json:"values,omitempty"`
}

// CustomPanel is a dashboard widget. Details is free-form JSON whose shape
// depends on Type (e.g. queries for line-chart). Group is recomputed by the
// server on save from the nearest preceding "row" panel.
type CustomPanel struct {
	UUID    string          `json:"uuid,omitempty"`
	Type    string          `json:"type,omitempty"`
	Title   string          `json:"title,omitempty"`
	Group   string          `json:"group,omitempty"`
	Details json.RawMessage `json:"details,omitempty"`
	Layout  PanelLayout     `json:"layout"`
}

type PanelLayout struct {
	W int    `json:"w"`
	H int    `json:"h"`
	X int    `json:"x"`
	Y int    `json:"y"`
	I string `json:"i"`
}

func (c *AxonopsHttpClient) dashboardTemplateURL(clusterType, clusterName string) string {
	return fmt.Sprintf("%s://%s/%s/dashboardtemplate/%s/%s/%s?dashver=%s", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid), esc(clusterType), esc(clusterName), dashboardTemplateVersion)
}

// GetDashboardTemplate returns the 2.0 dashboard template of a cluster. The
// server falls back to built-in defaults when none is stored, so the result
// is never empty for a known cluster.
func (c *AxonopsHttpClient) GetDashboardTemplate(ctx context.Context, clusterType, clusterName string) (*DashboardTemplate, error) {
	reqURL := c.dashboardTemplateURL(clusterType, clusterName)
	status, body, err := c.doJSON(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to get dashboard template: status %d for url %v, body: %s", status, reqURL, redactBody(body))
	}
	var tmpl DashboardTemplate
	if err := json.Unmarshal(body, &tmpl); err != nil {
		return nil, fmt.Errorf("failed to decode dashboard template: %w", err)
	}
	return &tmpl, nil
}

// SetDashboardTemplate replaces the whole 2.0 dashboard template of a cluster.
func (c *AxonopsHttpClient) SetDashboardTemplate(ctx context.Context, clusterType, clusterName string, tmpl DashboardTemplate) error {
	tmpl.Type = clusterType
	reqURL := c.dashboardTemplateURL(clusterType, clusterName)
	status, body, err := c.doJSON(ctx, "PUT", reqURL, tmpl)
	if err != nil {
		return err
	}
	if status < 200 || status > 299 {
		return fmt.Errorf("failed to set dashboard template: status %d for url %v, body: %s", status, reqURL, redactBody(body))
	}
	return nil
}

// FindCustomDashboard returns the dashboard with the given UUID, or nil.
func FindCustomDashboard(tmpl *DashboardTemplate, uuid string) *CustomDashboard {
	for i := range tmpl.Dashboards {
		if tmpl.Dashboards[i].UUID == uuid {
			return &tmpl.Dashboards[i]
		}
	}
	return nil
}
