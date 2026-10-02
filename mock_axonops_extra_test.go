package main

// Mock handlers for the 2.0 dashboard template, API token, Schema Registry
// config and Kafka broker endpoints. Routing lives in mock_axonops_test.go.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/google/uuid"
)

// builtinDashboardUUID is the dashboard the mock returns for a cluster with
// no stored template, like the server's built-in defaults.
const builtinDashboardUUID = "builtin-overview"

// handleDashboardTemplateV2 serves dashboardtemplate/{org}/{type}/{cluster}?dashver=2.0.
func (m *mockAxonOpsServer) handleDashboardTemplateV2(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	switch r.Method {
	case http.MethodGet:
		tmpl := m.dashboardsV2[key]
		if tmpl == nil {
			tmpl = &axonopsClient.DashboardTemplate{
				Type: clusterType,
				Dashboards: []axonopsClient.CustomDashboard{{
					UUID: builtinDashboardUUID,
					Name: "Overview",
					Panels: []axonopsClient.CustomPanel{{
						UUID: "builtin-row", Type: "row", Title: "Overview",
						Layout: axonopsClient.PanelLayout{W: 18, H: 1, I: "builtin-row"},
					}},
				}},
			}
		}
		writeJSON(w, http.StatusOK, tmpl)
	case http.MethodPut:
		var tmpl axonopsClient.DashboardTemplate
		if err := json.Unmarshal(readBody(r), &tmpl); err != nil || tmpl.Type == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Like the server: fill missing UUIDs and recompute row groups.
		for i := range tmpl.Dashboards {
			d := &tmpl.Dashboards[i]
			if d.UUID == "" {
				d.UUID = uuid.NewString()
			}
			group := ""
			for j := range d.Panels {
				p := &d.Panels[j]
				if p.UUID == "" {
					p.UUID = uuid.NewString()
				}
				if p.Type == "row" {
					p.Group = ""
					group = p.UUID
				} else {
					p.Group = group
				}
			}
		}
		m.dashboardsV2[key] = &tmpl
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// deleteDashboardOutOfBand removes a dashboard from the stored 2.0 template.
func (m *mockAxonOpsServer) deleteDashboardOutOfBand(clusterType, clusterName, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl := m.dashboardsV2[clusterKey(clusterType, clusterName)]
	if tmpl == nil {
		return
	}
	var kept []axonopsClient.CustomDashboard
	for _, d := range tmpl.Dashboards {
		if d.UUID != id {
			kept = append(kept, d)
		}
	}
	tmpl.Dashboards = kept
}

func (m *mockAxonOpsServer) dashboardV2(clusterType, clusterName, id string) *axonopsClient.CustomDashboard {
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl := m.dashboardsV2[clusterKey(clusterType, clusterName)]
	if tmpl == nil {
		return nil
	}
	return axonopsClient.FindCustomDashboard(tmpl, id)
}

// handleApiTokens serves {org}/createApiToken, {org}/listApiTokens and
// {org}/deleteApiToken. It reports false for any other path.
func (m *mockAxonOpsServer) handleApiTokens(w http.ResponseWriter, r *http.Request, action string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch action {
	case "createApiToken":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		var req axonopsClient.ApiToken
		if err := json.Unmarshal(readBody(r), &req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		secret := uuid.NewString()
		sum := sha256.Sum256([]byte(secret))
		token := axonopsClient.ApiToken{
			AllowedRoles: req.AllowedRoles,
			TokenExpiry:  req.TokenExpiry,
			KeyId:        uuid.NewString(),
			KeyHash:      hex.EncodeToString(sum[:]),
			CreationTime: int32(time.Now().Unix()),
		}
		m.apiTokens = append(m.apiTokens, token)
		writeJSON(w, http.StatusOK, axonopsClient.CreateApiTokenResponse{ApiKey: secret, ApiKeyId: token.KeyId})
	case "listApiTokens":
		// Like the server, an empty list is encoded as null.
		writeJSON(w, http.StatusOK, m.apiTokens)
	case "deleteApiToken":
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		var req struct {
			KeyHash string `json:"key_hash"`
		}
		_ = json.Unmarshal(readBody(r), &req)
		for i, t := range m.apiTokens {
			if t.KeyHash == req.KeyHash {
				m.apiTokens = append(m.apiTokens[:i], m.apiTokens[i+1:]...)
				w.WriteHeader(http.StatusOK)
				return true
			}
		}
		// The server answers 500 for an unknown hash.
		w.WriteHeader(http.StatusInternalServerError)
	default:
		return false
	}
	return true
}

func (m *mockAxonOpsServer) apiTokenExists(keyID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.apiTokens {
		if t.KeyId == keyID {
			return true
		}
	}
	return false
}

func (m *mockAxonOpsServer) apiTokenCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.apiTokens)
}

// handleSchemaRegistryConfig serves registry/configs[/{subject}].
func (m *mockAxonOpsServer) handleSchemaRegistryConfig(w http.ResponseWriter, r *http.Request, cluster string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subject := ""
	if len(rest) > 0 {
		subject = rest[0]
	}
	if m.srCompat[cluster] == nil {
		m.srCompat[cluster] = map[string]string{"": "BACKWARD"}
	}

	switch r.Method {
	case http.MethodGet:
		level, ok := m.srCompat[cluster][subject]
		status := http.StatusOK
		if subject != "" {
			// The server answers 201 for subject GETs.
			status = http.StatusCreated
		}
		if !ok {
			writeJSON(w, status, nil)
			return
		}
		writeJSON(w, status, map[string]string{"compatibilityLevel": level})
	case http.MethodPut:
		var req map[string]string
		if err := json.Unmarshal(readBody(r), &req); err != nil || req["compatibility"] == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		m.srCompat[cluster][subject] = req["compatibility"]
		writeJSON(w, http.StatusCreated, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *mockAxonOpsServer) schemaCompatibility(cluster, subject string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	level, ok := m.srCompat[cluster][subject]
	return level, ok
}

func (m *mockAxonOpsServer) deleteSchemaCompatibilityOutOfBand(cluster, subject string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.srCompat[cluster], subject)
}

func (m *mockAxonOpsServer) seedBroker(cluster string, broker axonopsClient.KafkaBrokerInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.brokers[cluster] == nil {
		m.brokers[cluster] = map[int64]*axonopsClient.KafkaBrokerInfo{}
	}
	m.brokers[cluster][int64(broker.BrokerID)] = &broker
}

// handleBroker serves broker/{brokerId}[?configNames=a,b].
func (m *mockAxonOpsServer) handleBroker(w http.ResponseWriter, r *http.Request, cluster, brokerID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.ParseInt(brokerID, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	broker := m.brokers[cluster][id]
	if broker == nil {
		// The server answers 500 when no Kafka node can serve the request.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	out := *broker
	if names := r.URL.Query().Get("configNames"); names != "" {
		want := map[string]bool{}
		for _, n := range strings.Split(names, ",") {
			want[n] = true
		}
		out.Configs = nil
		for _, c := range broker.Configs {
			if want[c.Name] {
				out.Configs = append(out.Configs, c)
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// dashboardsV2List returns a copy of the stored 2.0 dashboards of a cluster.
func (m *mockAxonOpsServer) dashboardsV2List(clusterType, clusterName string) []axonopsClient.CustomDashboard {
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl := m.dashboardsV2[clusterKey(clusterType, clusterName)]
	if tmpl == nil {
		return nil
	}
	return append([]axonopsClient.CustomDashboard(nil), tmpl.Dashboards...)
}
