package main

// mock_axonops_test.go implements a small stateful fake of the AxonOps HTTP
// API (net/http/httptest) covering exactly the endpoints exercised by
// client/http_client.go and client/delete_retry.go. It backs the acceptance
// tests in this package, which drive the real provider (via
// terraform-plugin-testing) against this fake instead of a live AxonOps
// server.
//
// State is held in-memory, guarded by a single mutex. Helper methods are
// provided so tests can mutate state out-of-band (simulating drift / manual
// deletion via the AxonOps UI) between Terraform plan/apply steps.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/google/uuid"
)

// mockIntegration mirrors axonopsClient.IntegrationDefinition but keeps the
// real (unmasked) secret value server-side so the mock can return a masked
// value on GET, the way the real AxonOps API does for sensitive params.
type mockIntegration struct {
	ID     string
	Type   string
	Params map[string]string
	// hiddenReads is how many more list GETs omit this integration,
	// simulating an API that acknowledges a write before reads see it.
	hiddenReads int
}

// maskedIntegrationParamKeys lists which Params keys are masked on GET for
// each integration type. Only fields the provider treats as `Sensitive` in
// its schema are masked here.
var maskedIntegrationParamKeys = map[string][]string{
	"slack":      {"url"},
	"teams":      {"url"},
	"pagerduty":  {"integrationKey"},
	"opsgenie":   {"key"},
	"servicenow": {"password"},
}

const maskedSecretValue = "**MASKED**"

type routingEntry struct {
	Type            string
	Routing         []axonopsClient.IntegrationRoute
	OverrideInfo    bool
	OverrideWarning bool
	OverrideError   bool
}

type mockSchemaVersion struct {
	id         int
	version    int
	schema     string
	schemaType string
}

type mockAxonOpsServer struct {
	mu sync.Mutex

	orgID  string
	server *httptest.Server

	// kafka topics: cluster -> topic name -> topic
	topics map[string]map[string]*axonopsClient.TopicInfo
	// kafka acls: cluster -> list of ACLResource
	acls map[string][]axonopsClient.ACLResource
	// kafka connect: cluster -> connectCluster -> connector name -> connector
	connectors map[string]map[string]map[string]*axonopsClient.KafkaConnectorResponse
	// schema registry: cluster -> subject -> ordered list of versions (1-indexed by position)
	schemas map[string]map[string][]*mockSchemaVersion

	// logcollectors: "clusterType/clusterName" -> list
	logCollectors map[string][]axonopsClient.LogCollectorConfig
	// healthchecks: "clusterType/clusterName" -> document
	healthchecks map[string]*axonopsClient.HealthchecksResponse
	// adaptive repair: "clusterType/clusterName" -> settings
	adaptiveRepair map[string]*axonopsClient.AdaptiveRepairSettings
	// cassandra backups: "clusterType/clusterName" -> list of (id, backup)
	cassandraBackups map[string][]axonopsClient.CassandraBackup
	// scheduled repairs: clusterName -> list of entries
	scheduledRepairs map[string][]axonopsClient.ScheduledRepairEntry

	// dashboard templates: "clusterType/clusterName" -> templates (test fixtures)
	dashboards map[string]*axonopsClient.DashboardTemplateResponse

	// alert rules (metric + log share the same store): "clusterType/clusterName" -> rules
	alertRules map[string][]axonopsClient.MetricAlertRule

	// integrations: "clusterType/clusterName" -> definitions
	integrations map[string][]*mockIntegration
	// routings: "clusterType/clusterName" -> routing entries keyed by Type
	routings map[string]map[string]*routingEntry

	// silences: "clusterType/clusterName" -> list
	silences map[string][]axonopsClient.SilenceWindow

	// 2.0 dashboard templates (?dashver=2.0): "clusterType/clusterName" -> template
	dashboardsV2 map[string]*axonopsClient.DashboardTemplate
	// API tokens of the org
	apiTokens []axonopsClient.ApiToken
	// schema registry compatibility: cluster -> subject ("" = global) -> level
	srCompat map[string]map[string]string
	// kafka brokers: cluster -> broker ID -> broker
	brokers map[string]map[int64]*axonopsClient.KafkaBrokerInfo
	// cluster inventory (test fixtures): org -> clusterType -> cluster names/status
	orgClusters map[string]map[string]map[string]int
	// nodes: "clusterType/clusterName" -> nodes
	nodes map[string][]axonopsClient.ClusterNodeInfo
	// keyspaces: "clusterType/clusterName" -> keyspaces
	keyspaces map[string][]axonopsClient.CassandraKeyspace
	// soft-deleted schema subjects: cluster -> subject -> true
	deletedSubjects map[string]map[string]bool

	// integrationReadLag is how many list GETs omit a newly created
	// integration. Zero (the default) makes writes visible immediately.
	integrationReadLag int
}

func newMockAxonOpsServer(t interface{ Cleanup(func()) }) *mockAxonOpsServer {
	m := &mockAxonOpsServer{
		orgID:            "testorg",
		topics:           map[string]map[string]*axonopsClient.TopicInfo{},
		connectors:       map[string]map[string]map[string]*axonopsClient.KafkaConnectorResponse{},
		schemas:          map[string]map[string][]*mockSchemaVersion{},
		logCollectors:    map[string][]axonopsClient.LogCollectorConfig{},
		healthchecks:     map[string]*axonopsClient.HealthchecksResponse{},
		adaptiveRepair:   map[string]*axonopsClient.AdaptiveRepairSettings{},
		cassandraBackups: map[string][]axonopsClient.CassandraBackup{},
		scheduledRepairs: map[string][]axonopsClient.ScheduledRepairEntry{},
		dashboards:       map[string]*axonopsClient.DashboardTemplateResponse{},
		alertRules:       map[string][]axonopsClient.MetricAlertRule{},
		integrations:     map[string][]*mockIntegration{},
		routings:         map[string]map[string]*routingEntry{},
		silences:         map[string][]axonopsClient.SilenceWindow{},
		acls:             map[string][]axonopsClient.ACLResource{},
		dashboardsV2:     map[string]*axonopsClient.DashboardTemplate{},
		srCompat:         map[string]map[string]string{},
		brokers:          map[string]map[int64]*axonopsClient.KafkaBrokerInfo{},
		orgClusters:      map[string]map[string]map[string]int{},
		nodes:            map[string][]axonopsClient.ClusterNodeInfo{},
		keyspaces:        map[string][]axonopsClient.CassandraKeyspace{},
		deletedSubjects:  map[string]map[string]bool{},
	}

	mux := http.NewServeMux()
	m.registerRoutes(mux)
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)

	return m
}

func (m *mockAxonOpsServer) URL() string {
	return strings.TrimPrefix(m.server.URL, "http://")
}

func clusterKey(clusterType, clusterName string) string {
	return clusterType + "/" + clusterName
}

// --- test helpers for out-of-band mutation (simulate drift) ---

func (m *mockAxonOpsServer) deleteTopicOutOfBand(cluster, topic string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.topics[cluster], topic)
}

func (m *mockAxonOpsServer) deleteACLOutOfBand(cluster string, acl axonopsClient.KafkaACL) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var resources []axonopsClient.ACLResource
	for _, res := range m.acls[cluster] {
		var kept []axonopsClient.KafkaACL
		for _, a := range res.ACLs {
			if a == acl && res.ResourceName == acl.ResourceName {
				continue
			}
			kept = append(kept, a)
		}
		if len(kept) > 0 {
			res.ACLs = kept
			resources = append(resources, res)
		}
	}
	m.acls[cluster] = resources
}

func (m *mockAxonOpsServer) seedDashboard(clusterType, clusterName string, dash axonopsClient.Dashboard) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)
	if m.dashboards[key] == nil {
		m.dashboards[key] = &axonopsClient.DashboardTemplateResponse{}
	}
	m.dashboards[key].Dashboards = append(m.dashboards[key].Dashboards, dash)
}

// setIntegrationReadLag makes each integration created from now on invisible
// to the next n list GETs.
func (m *mockAxonOpsServer) setIntegrationReadLag(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.integrationReadLag = n
}

// --- routing ---

// seedCluster registers a cluster in the /orgs hierarchy of org with the
// given alert status (0 green, 1 amber, 2 red).
func (m *mockAxonOpsServer) seedCluster(org, clusterType, clusterName string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.orgClusters[org] == nil {
		m.orgClusters[org] = map[string]map[string]int{}
	}
	if m.orgClusters[org][clusterType] == nil {
		m.orgClusters[org][clusterType] = map[string]int{}
	}
	m.orgClusters[org][clusterType][clusterName] = status
}

func (m *mockAxonOpsServer) seedNodes(clusterType, clusterName string, nodes ...axonopsClient.ClusterNodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)
	m.nodes[key] = append(m.nodes[key], nodes...)
}

func (m *mockAxonOpsServer) seedKeyspaces(clusterType, clusterName string, keyspaces ...axonopsClient.CassandraKeyspace) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)
	m.keyspaces[key] = append(m.keyspaces[key], keyspaces...)
}

// softDeleteSubjectOutOfBand marks a subject as soft-deleted: it disappears
// from the subject list unless addDeleted=true is passed.
func (m *mockAxonOpsServer) softDeleteSubjectOutOfBand(cluster, subject string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deletedSubjects[cluster] == nil {
		m.deletedSubjects[cluster] = map[string]bool{}
	}
	m.deletedSubjects[cluster][subject] = true
}

func (m *mockAxonOpsServer) handleOrgs(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	type clusterNode struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	type typeNode struct {
		Name     string        `json:"name"`
		Type     string        `json:"type"`
		Children []clusterNode `json:"children"`
	}
	type orgNode struct {
		Name     string     `json:"name"`
		Type     string     `json:"type"`
		Children []typeNode `json:"children"`
	}
	orgs := []orgNode{}
	for org, types := range m.orgClusters {
		on := orgNode{Name: org, Type: "org"}
		for clusterType, clusters := range types {
			tn := typeNode{Name: clusterType, Type: "type"}
			for name, status := range clusters {
				tn.Children = append(tn.Children, clusterNode{Name: name, Type: clusterType, Status: status})
			}
			on.Children = append(on.Children, tn)
		}
		orgs = append(orgs, on)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"children": orgs})
}

func (m *mockAxonOpsServer) handleNodes(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	nodes := m.nodes[clusterKey(clusterType, clusterName)]
	if nodes == nil {
		nodes = []axonopsClient.ClusterNodeInfo{}
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (m *mockAxonOpsServer) handleKeyspaces(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keyspaces := m.keyspaces[clusterKey(clusterType, clusterName)]
	if keyspaces == nil {
		keyspaces = []axonopsClient.CassandraKeyspace{}
	}
	writeJSON(w, http.StatusOK, keyspaces)
}

func (m *mockAxonOpsServer) registerRoutes(mux *http.ServeMux) {
	// SAML probe: always 404 so the provider uses the non-SAML host layout.
	mux.HandleFunc("/dashboard/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/api/v1/", m.handleAPI)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readBody(r *http.Request) []byte {
	b, _ := io.ReadAll(r.Body)
	return b
}

// handleAPI dispatches on the URL path beneath /api/v1/. Two distinct URL
// shapes are used by client/http_client.go:
//
//   - org-first: /api/v1/{org}/{resource}/... (only the kafka.* endpoints,
//     e.g. /api/v1/{org}/kafka/{cluster}/topics)
//   - literal-first: /api/v1/{resource}/{org}/... (everything else, e.g.
//     /api/v1/silenceWindow/{org}/{clusterType}/{clusterName},
//     /api/v1/logcollectors/{org}/{clusterType}/{clusterName})
//
// This split exactly mirrors the fmt.Sprintf patterns in http_client.go; see
// the comment on each handler below for the specific endpoint it serves.
func (m *mockAxonOpsServer) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch parts[0] {
	case "orgs":
		m.handleOrgs(w, r)
		return
	case "nodes":
		// nodes/{org}/{clusterType}/{clusterName}
		if len(parts) >= 4 {
			m.handleNodes(w, r, parts[2], parts[3])
			return
		}
	case "keyspaces":
		// keyspaces/{org}/{clusterType}/{clusterName}
		if len(parts) >= 4 {
			m.handleKeyspaces(w, r, parts[2], parts[3])
			return
		}
	case "logcollectors":
		// logcollectors/{org}/{clusterType}/{clusterName}
		if len(parts) >= 4 {
			m.handleLogCollectors(w, r, parts[2], parts[3])
			return
		}
	case "healthchecks":
		// healthchecks/{org}/{clusterType}/{clusterName}
		if len(parts) >= 4 {
			m.handleHealthchecks(w, r, parts[2], parts[3])
			return
		}
	case "adaptiveRepair":
		// adaptiveRepair/{org}/{clusterType}/{clusterName}
		if len(parts) >= 4 {
			m.handleAdaptiveRepair(w, r, parts[2], parts[3])
			return
		}
	case "cassandraScheduleSnapshot":
		// cassandraScheduleSnapshot/{org}/{clusterType}/{clusterName}: used
		// for both GetCassandraBackups (GET) and DeleteCassandraBackup (DELETE).
		if len(parts) >= 4 {
			m.handleCassandraScheduleSnapshot(w, r, parts[2], parts[3])
			return
		}
	case "cassandraSnapshot":
		// cassandraSnapshot/{org}/{clusterType}/{clusterName}: CreateCassandraBackup.
		if len(parts) >= 4 {
			m.handleCreateCassandraBackup(w, r, parts[2], parts[3])
			return
		}
	case "alert-rules":
		// alert-rules/{org}/{clusterType}/{clusterName}[/{id}]
		if len(parts) >= 4 {
			m.handleAlertRules(w, r, parts[2], parts[3], parts[4:])
			return
		}
	case "dashboardtemplate":
		// dashboardtemplate/{org}/{clusterType}/{clusterName}
		if len(parts) >= 4 {
			m.handleDashboardTemplates(w, r, parts[2], parts[3])
			return
		}
	case "integrations":
		// integrations/{org}/{clusterType}/{clusterName}[/{id}]
		if len(parts) >= 4 {
			m.handleIntegrations(w, r, parts[2], parts[3], parts[4:])
			return
		}
	case "integrations-override":
		// integrations-override/{org}/{clusterType}/{clusterName}/{routeType}/{severity}
		if len(parts) >= 6 {
			m.handleIntegrationOverride(w, r, parts[2], parts[3], parts[4], parts[5])
			return
		}
	case "integrations-routing":
		// integrations-routing/{org}/{clusterType}/{clusterName}/{routeType}/{severity}/{integrationID}
		if len(parts) >= 7 {
			m.handleIntegrationRouting(w, r, parts[2], parts[3], parts[4], parts[5], parts[6])
			return
		}
	case "repair":
		// repair/{org}/cassandra/{clusterName}: GetScheduledRepairs.
		if len(parts) >= 4 {
			m.handleGetScheduledRepairs(w, r, parts[3])
			return
		}
	case "addrepair":
		// addrepair/{org}/cassandra/{clusterName}: CreateScheduledRepair.
		if len(parts) >= 4 {
			m.handleCreateScheduledRepair(w, r, parts[3])
			return
		}
	case "cassandrascheduledrepair":
		// cassandrascheduledrepair/{org}/cassandra/{clusterName}?id=...: DeleteScheduledRepair.
		if len(parts) >= 4 {
			m.handleDeleteScheduledRepair(w, r, parts[3])
			return
		}
	case "silenceWindow":
		// silenceWindow/{org}/{clusterType}/{clusterName}[/{id}]
		if len(parts) >= 4 {
			m.handleSilence(w, r, parts[2], parts[3], parts[4:])
			return
		}
	default:
		// org-first: {org}/createApiToken, {org}/listApiTokens, {org}/deleteApiToken
		if len(parts) == 2 && m.handleApiTokens(w, r, parts[1]) {
			return
		}
		// org-first: {org}/kafka/...
		if len(parts) >= 2 && parts[1] == "kafka" {
			m.handleKafka(w, r, parts[2:])
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

// --- Kafka: topics, acls, connect, schema registry ---

func (m *mockAxonOpsServer) handleKafka(w http.ResponseWriter, r *http.Request, rest []string) {
	// rest: {clusterName}/{resource}/...
	if len(rest) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	cluster := rest[0]
	switch rest[1] {
	case "topics":
		m.handleTopics(w, r, cluster, rest[2:])
	case "acls":
		m.handleACLs(w, r, cluster)
	case "connect":
		// connect/{connectCluster}/...
		if len(rest) < 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		m.handleConnect(w, r, cluster, rest[2], rest[3:])
	case "broker":
		// broker/{brokerId}
		if len(rest) != 3 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		m.handleBroker(w, r, cluster, rest[2])
	case "registry":
		// registry/configs[/{subject}]
		if len(rest) >= 3 && rest[2] == "configs" {
			m.handleSchemaRegistryConfig(w, r, cluster, rest[3:])
			return
		}
		// registry/subjects[/{subject}[/{version}]]
		if len(rest) == 3 && rest[2] == "subjects" {
			m.handleSchemaSubjects(w, r, cluster)
			return
		}
		if len(rest) < 4 || rest[2] != "subjects" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		m.handleSchemaRegistry(w, r, cluster, rest[3:])
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *mockAxonOpsServer) handleTopics(w http.ResponseWriter, r *http.Request, cluster string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.topics[cluster] == nil {
		m.topics[cluster] = map[string]*axonopsClient.TopicInfo{}
	}

	if len(rest) == 0 {
		switch r.Method {
		case http.MethodPost:
			var payload axonopsClient.KafkaTopic
			_ = json.Unmarshal(readBody(r), &payload)
			m.topics[cluster][payload.TopicName] = &axonopsClient.TopicInfo{
				Name:              payload.TopicName,
				Partitions:        payload.PartitionCount,
				ReplicationFactor: payload.ReplicationFactor,
				Config:            payload.Configs,
			}
			w.WriteHeader(http.StatusCreated)
			return
		case http.MethodGet:
			var all []axonopsClient.TopicInfo
			for _, t := range m.topics[cluster] {
				all = append(all, *t)
			}
			writeJSON(w, http.StatusOK, all)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	topicName, err := url.PathUnescape(rest[0])
	if err != nil {
		topicName = rest[0]
	}

	if len(rest) == 1 {
		switch r.Method {
		case http.MethodGet:
			t, ok := m.topics[cluster][topicName]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, t)
			return
		case http.MethodDelete:
			if _, ok := m.topics[cluster][topicName]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(m.topics[cluster], topicName)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// rest[1] is a subresource: configs | partitions | replicationfactor
	t, ok := m.topics[cluster][topicName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch rest[1] {
	case "configs":
		switch r.Method {
		case http.MethodGet:
			entries := make([]axonopsClient.TopicConfigEntry, 0, len(t.Config))
			for _, c := range t.Config {
				entries = append(entries, axonopsClient.TopicConfigEntry{
					Name: c.Name, Value: c.Value, Source: "DYNAMIC_TOPIC_CONFIG", IsExplicitlySet: true,
				})
			}
			resp := axonopsClient.TopicConfigResponse{
				TopicDescription: []axonopsClient.TopicConfigDescription{
					{TopicName: topicName, ConfigEntries: entries},
				},
			}
			writeJSON(w, http.StatusOK, resp)
			return
		case http.MethodPut:
			var wrapper axonopsClient.ConfigsWrapper
			_ = json.Unmarshal(readBody(r), &wrapper)
			byName := map[string]string{}
			for _, c := range t.Config {
				byName[c.Name] = c.Value
			}
			for _, c := range wrapper.Configs {
				if c.Op == "DELETE" {
					delete(byName, c.Key)
				} else {
					byName[c.Key] = c.Value
				}
			}
			var newConfig []axonopsClient.KafkaTopicConfig
			for k, v := range byName {
				newConfig = append(newConfig, axonopsClient.KafkaTopicConfig{Name: k, Value: v})
			}
			t.Config = newConfig
			w.WriteHeader(http.StatusNoContent)
			return
		}
	case "partitions":
		if r.Method == http.MethodPut {
			var payload map[string]int32
			_ = json.Unmarshal(readBody(r), &payload)
			t.Partitions = payload["partitions"]
			w.WriteHeader(http.StatusOK)
			return
		}
	case "replicationfactor":
		if r.Method == http.MethodPut {
			var payload map[string]string
			_ = json.Unmarshal(readBody(r), &payload)
			if v, err := strconv.Atoi(payload["replication_factor"]); err == nil {
				t.ReplicationFactor = int32(v)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (m *mockAxonOpsServer) handleACLs(w http.ResponseWriter, r *http.Request, cluster string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, axonopsClient.ACLResponse{ACLResources: m.acls[cluster]})
	case http.MethodPost:
		var acl axonopsClient.KafkaACL
		_ = json.Unmarshal(readBody(r), &acl)
		m.addACL(cluster, acl)
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		var acl axonopsClient.KafkaACL
		_ = json.Unmarshal(readBody(r), &acl)
		m.removeACL(cluster, acl)
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *mockAxonOpsServer) addACL(cluster string, acl axonopsClient.KafkaACL) {
	for i, res := range m.acls[cluster] {
		if res.ResourceType == acl.ResourceType && res.ResourceName == acl.ResourceName && res.ResourcePatternType == acl.ResourcePatternType {
			m.acls[cluster][i].ACLs = append(m.acls[cluster][i].ACLs, acl)
			return
		}
	}
	m.acls[cluster] = append(m.acls[cluster], axonopsClient.ACLResource{
		ResourceType:        acl.ResourceType,
		ResourceName:        acl.ResourceName,
		ResourcePatternType: acl.ResourcePatternType,
		ACLs:                []axonopsClient.KafkaACL{acl},
	})
}

func (m *mockAxonOpsServer) removeACL(cluster string, acl axonopsClient.KafkaACL) {
	var resources []axonopsClient.ACLResource
	for _, res := range m.acls[cluster] {
		if res.ResourceType != acl.ResourceType || res.ResourceName != acl.ResourceName || res.ResourcePatternType != acl.ResourcePatternType {
			resources = append(resources, res)
			continue
		}
		var kept []axonopsClient.KafkaACL
		for _, a := range res.ACLs {
			if a == acl {
				continue
			}
			kept = append(kept, a)
		}
		if len(kept) > 0 {
			res.ACLs = kept
			resources = append(resources, res)
		}
	}
	m.acls[cluster] = resources
}

func (m *mockAxonOpsServer) handleConnect(w http.ResponseWriter, r *http.Request, cluster, connectCluster string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connectors[cluster] == nil {
		m.connectors[cluster] = map[string]map[string]*axonopsClient.KafkaConnectorResponse{}
	}
	if m.connectors[cluster][connectCluster] == nil {
		m.connectors[cluster][connectCluster] = map[string]*axonopsClient.KafkaConnectorResponse{}
	}
	store := m.connectors[cluster][connectCluster]

	if len(rest) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch rest[0] {
	case "connector":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload axonopsClient.KafkaConnector
		_ = json.Unmarshal(readBody(r), &payload)
		cfg := map[string]string{}
		for k, v := range payload.Config {
			cfg[k] = v
		}
		// Simulate server-injected config keys: Kafka Connect always adds
		// "name" to the effective config.
		cfg["name"] = payload.Name
		connType := "sink"
		if strings.Contains(strings.ToLower(cfg["connector.class"]), "source") {
			connType = "source"
		}
		result := &axonopsClient.KafkaConnectorResponse{Name: payload.Name, Config: cfg, Type: connType}
		store[payload.Name] = result
		writeJSON(w, http.StatusCreated, result)
		return
	case "connectors":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		entries := map[string]axonopsClient.ConnectorListEntry{}
		for name, c := range store {
			entries[name] = axonopsClient.ConnectorListEntry{
				Info: *c,
				Status: axonopsClient.ConnectorStatus{
					Name:      name,
					Connector: axonopsClient.ConnectorStateInfo{State: "RUNNING"},
					Type:      c.Type,
				},
			}
		}
		writeJSON(w, http.StatusOK, axonopsClient.ConnectorsListResponse{
			ClusterName: connectCluster, Connectors: entries,
		})
		return
	default:
		// {connectorName}/config (PUT) or {connectorName} (DELETE)
		connectorName := rest[0]
		if len(rest) == 2 && rest[1] == "config" {
			if r.Method != http.MethodPut {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			c, ok := store[connectorName]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var payload axonopsClient.KafkaConnectorConfig
			_ = json.Unmarshal(readBody(r), &payload)
			newCfg := map[string]string{}
			for k, v := range payload.Config {
				newCfg[k] = v
			}
			newCfg["name"] = connectorName
			c.Config = newCfg
			writeJSON(w, http.StatusOK, c)
			return
		}
		if len(rest) == 1 {
			if r.Method != http.MethodDelete {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if _, ok := store[connectorName]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(store, connectorName)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *mockAxonOpsServer) handleSchemaSubjects(w http.ResponseWriter, r *http.Request, cluster string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	addDeleted := r.URL.Query().Get("addDeleted") == "true"
	subjects := []string{}
	for subject := range m.schemas[cluster] {
		if m.deletedSubjects[cluster][subject] && !addDeleted {
			continue
		}
		subjects = append(subjects, subject)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"subjects": subjects, "Configs": map[string]string{}})
}

func (m *mockAxonOpsServer) handleSchemaRegistry(w http.ResponseWriter, r *http.Request, cluster string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(rest) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	subject := rest[0]
	if m.schemas[cluster] == nil {
		m.schemas[cluster] = map[string][]*mockSchemaVersion{}
	}

	if len(rest) == 1 {
		switch r.Method {
		case http.MethodPost:
			var payload axonopsClient.CreateSchemaRequest
			_ = json.Unmarshal(readBody(r), &payload)
			versions := m.schemas[cluster][subject]
			newVersion := &mockSchemaVersion{
				id:         len(versions) + 1000,
				version:    len(versions) + 1,
				schema:     payload.Schema,
				schemaType: payload.SchemaType,
			}
			m.schemas[cluster][subject] = append(versions, newVersion)
			writeJSON(w, http.StatusCreated, axonopsClient.CreateSchemaResponse{Id: newVersion.id})
			return
		case http.MethodDelete:
			delete(m.schemas[cluster], subject)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// GET /{subject}/{version}
	versions := m.schemas[cluster][subject]
	if len(versions) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var v *mockSchemaVersion
	if rest[1] == "latest" {
		v = versions[len(versions)-1]
	} else if n, err := strconv.Atoi(rest[1]); err == nil {
		for _, cand := range versions {
			if cand.version == n {
				v = cand
				break
			}
		}
	}
	if v == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, axonopsClient.SchemaRegistryVersionedSchema{
		Id: v.id, Version: v.version, Schema: v.schema, Type: v.schemaType,
	})
}

// --- log collectors / healthchecks (non-versioned /api/v1/... paths) ---

func (m *mockAxonOpsServer) handleLogCollectors(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, m.logCollectors[key])
	case http.MethodPut:
		body := readBody(r)
		vals, err := url.ParseQuery(string(body))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		raw := vals.Get("addlogs")
		var collectors []axonopsClient.LogCollectorConfig
		if err := json.Unmarshal([]byte(raw), &collectors); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.logCollectors[key] = collectors
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *mockAxonOpsServer) handleHealthchecks(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	switch r.Method {
	case http.MethodGet:
		doc := m.healthchecks[key]
		if doc == nil {
			doc = &axonopsClient.HealthchecksResponse{}
		}
		writeJSON(w, http.StatusOK, doc)
	case http.MethodPut:
		var doc axonopsClient.HealthchecksResponse
		if err := json.Unmarshal(readBody(r), &doc); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.healthchecks[key] = &doc
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// --- adaptive repair ---

func (m *mockAxonOpsServer) handleAdaptiveRepair(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	switch r.Method {
	case http.MethodGet:
		s := m.adaptiveRepair[key]
		if s == nil {
			s = &axonopsClient.AdaptiveRepairSettings{}
		}
		writeJSON(w, http.StatusOK, s)
	case http.MethodPost:
		var s axonopsClient.AdaptiveRepairSettings
		if err := json.Unmarshal(readBody(r), &s); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.adaptiveRepair[key] = &s
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// --- cassandra backups ---
// The real API stores backups inside a "ScheduledSnapshots" envelope whose
// Params field is a JSON-encoded array of {"BackupDetails": "<json string>"}.
// We replicate that shape so client.GetCassandraBackups' parsing logic is
// exercised faithfully.

// handleCassandraScheduleSnapshot serves the "cassandraScheduleSnapshot"
// endpoint, which client.GetCassandraBackups (GET) and
// client.DeleteCassandraBackup (DELETE) both target.
func (m *mockAxonOpsServer) handleCassandraScheduleSnapshot(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	switch r.Method {
	case http.MethodGet:
		var snapshots []axonopsClient.CassandraScheduledSnapshot
		for _, b := range m.cassandraBackups[key] {
			// The real API does not return remoteConfig as it was sent.
			if b.RemoteConfig != "" {
				b.RemoteConfig = maskedSecretValue
			}
			detailsJSON, _ := json.Marshal(b)
			params := []axonopsClient.CassandraScheduledParam{{BackupDetails: string(detailsJSON)}}
			paramsJSON, _ := json.Marshal(params)
			snapshots = append(snapshots, axonopsClient.CassandraScheduledSnapshot{
				ID:     b.ID,
				Params: paramsJSON,
			})
		}
		writeJSON(w, http.StatusOK, axonopsClient.CassandraBackupsResponse{ScheduledSnapshots: snapshots})
	case http.MethodDelete:
		var ids []string
		_ = json.Unmarshal(readBody(r), &ids)
		idSet := map[string]bool{}
		for _, id := range ids {
			idSet[id] = true
		}
		var kept []axonopsClient.CassandraBackup
		for _, b := range m.cassandraBackups[key] {
			if !idSet[b.ID] {
				kept = append(kept, b)
			}
		}
		m.cassandraBackups[key] = kept
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleCreateCassandraBackup serves the "cassandraSnapshot" endpoint
// (client.CreateCassandraBackup, POST only).
func (m *mockAxonOpsServer) handleCreateCassandraBackup(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	var b axonopsClient.CassandraBackup
	if err := json.Unmarshal(readBody(r), &b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	m.cassandraBackups[key] = append(m.cassandraBackups[key], b)
	w.WriteHeader(http.StatusCreated)
}

// --- scheduled repairs ---

func (m *mockAxonOpsServer) handleGetScheduledRepairs(w http.ResponseWriter, r *http.Request, clusterName string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var raw struct {
		ScheduledRepairs []struct {
			ID     string          `json:"ID"`
			Params json.RawMessage `json:"Params"`
		} `json:"ScheduledRepairs"`
	}
	for _, entry := range m.scheduledRepairs[clusterName] {
		paramsJSON, _ := json.Marshal(entry.Params)
		raw.ScheduledRepairs = append(raw.ScheduledRepairs, struct {
			ID     string          `json:"ID"`
			Params json.RawMessage `json:"Params"`
		}{ID: entry.ID, Params: paramsJSON})
	}
	writeJSON(w, http.StatusOK, raw)
}

func (m *mockAxonOpsServer) handleCreateScheduledRepair(w http.ResponseWriter, r *http.Request, clusterName string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var params axonopsClient.ScheduledRepairParams
	if err := json.Unmarshal(readBody(r), &params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	entry := axonopsClient.ScheduledRepairEntry{
		ID:     uuid.New().String(),
		Params: []axonopsClient.ScheduledRepairParams{params},
	}
	m.scheduledRepairs[clusterName] = append(m.scheduledRepairs[clusterName], entry)
	w.WriteHeader(http.StatusCreated)
}

func (m *mockAxonOpsServer) handleDeleteScheduledRepair(w http.ResponseWriter, r *http.Request, clusterName string) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	m.mu.Lock()
	defer m.mu.Unlock()

	var kept []axonopsClient.ScheduledRepairEntry
	for _, e := range m.scheduledRepairs[clusterName] {
		if e.ID != id {
			kept = append(kept, e)
		}
	}
	m.scheduledRepairs[clusterName] = kept
	w.WriteHeader(http.StatusOK)
}

// --- dashboard templates ---

func (m *mockAxonOpsServer) handleDashboardTemplates(w http.ResponseWriter, r *http.Request, clusterType, clusterName string) {
	if r.URL.Query().Get("dashver") == "2.0" {
		m.handleDashboardTemplateV2(w, r, clusterType, clusterName)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)
	doc := m.dashboards[key]
	if doc == nil {
		doc = &axonopsClient.DashboardTemplateResponse{}
	}
	writeJSON(w, http.StatusOK, doc)
}

// --- alert rules (metric + log) ---

func (m *mockAxonOpsServer) handleAlertRules(w http.ResponseWriter, r *http.Request, clusterType, clusterName string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, axonopsClient.AlertRulesResponse{MetricRules: m.alertRules[key]})
			return
		case http.MethodPost:
			var rule axonopsClient.MetricAlertRule
			if err := json.Unmarshal(readBody(r), &rule); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			rules := m.alertRules[key]
			updated := false
			for i, existing := range rules {
				if existing.ID == rule.ID && rule.ID != "" {
					rules[i] = rule
					updated = true
					break
				}
			}
			if !updated {
				// The real API ignores client-supplied IDs and always
				// assigns its own server-side ID.
				rule.ID = uuid.New().String()
				rules = append(rules, rule)
			}
			m.alertRules[key] = rules
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// DELETE /{id}
	if r.Method == http.MethodDelete {
		id := rest[0]
		var kept []axonopsClient.MetricAlertRule
		for _, rule := range m.alertRules[key] {
			if rule.ID != id {
				kept = append(kept, rule)
			}
		}
		m.alertRules[key] = kept
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// --- integrations & routing ---

func (m *mockAxonOpsServer) maskedParams(def *mockIntegration) map[string]string {
	out := make(map[string]string, len(def.Params))
	for k, v := range def.Params {
		out[k] = v
	}
	for _, sensitiveKey := range maskedIntegrationParamKeys[def.Type] {
		if _, ok := out[sensitiveKey]; ok {
			out[sensitiveKey] = maskedSecretValue
		}
	}
	return out
}

func (m *mockAxonOpsServer) handleIntegrations(w http.ResponseWriter, r *http.Request, clusterType, clusterName string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			var defs []axonopsClient.IntegrationDefinition
			for _, d := range m.integrations[key] {
				if d.hiddenReads > 0 {
					d.hiddenReads--
					continue
				}
				defs = append(defs, axonopsClient.IntegrationDefinition{
					ID: d.ID, Type: d.Type, Params: m.maskedParams(d),
				})
			}
			var routings []axonopsClient.IntegrationRouting
			for _, routing := range m.routings[key] {
				routings = append(routings, axonopsClient.IntegrationRouting{
					Type: routing.Type, Routing: routing.Routing,
					OverrideInfo: routing.OverrideInfo, OverrideWarning: routing.OverrideWarning, OverrideError: routing.OverrideError,
				})
			}
			writeJSON(w, http.StatusOK, axonopsClient.IntegrationsResponse{Definitions: defs, Routings: routings})
			return
		case http.MethodPost:
			var payload axonopsClient.IntegrationPayload
			if err := json.Unmarshal(readBody(r), &payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if payload.ID != "" {
				for _, d := range m.integrations[key] {
					if d.ID == payload.ID {
						d.Type = payload.Type
						d.Params = payload.Params
						w.WriteHeader(http.StatusOK)
						return
					}
				}
			}
			newDef := &mockIntegration{ID: uuid.New().String(), Type: payload.Type, Params: payload.Params, hiddenReads: m.integrationReadLag}
			m.integrations[key] = append(m.integrations[key], newDef)
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// DELETE /{id}
	if r.Method == http.MethodDelete {
		id := rest[0]
		var kept []*mockIntegration
		for _, d := range m.integrations[key] {
			if d.ID != id {
				kept = append(kept, d)
			}
		}
		m.integrations[key] = kept
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (m *mockAxonOpsServer) routingEntryFor(key, routeType string) *routingEntry {
	if m.routings[key] == nil {
		m.routings[key] = map[string]*routingEntry{}
	}
	decoded, _ := url.QueryUnescape(routeType)
	decoded = strings.ReplaceAll(decoded, "%20", " ")
	re, ok := m.routings[key][decoded]
	if !ok {
		re = &routingEntry{Type: decoded}
		m.routings[key][decoded] = re
	}
	return re
}

func (m *mockAxonOpsServer) handleIntegrationOverride(w http.ResponseWriter, r *http.Request, clusterType, clusterName, routeType, severity string) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	var payload axonopsClient.OverridePayload
	if err := json.Unmarshal(readBody(r), &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	re := m.routingEntryFor(key, routeType)
	switch strings.ToLower(severity) {
	case "info":
		re.OverrideInfo = payload.Value
	case "warning":
		re.OverrideWarning = payload.Value
	case "error":
		re.OverrideError = payload.Value
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *mockAxonOpsServer) handleIntegrationRouting(w http.ResponseWriter, r *http.Request, clusterType, clusterName, routeType, severity, integrationID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)
	re := m.routingEntryFor(key, routeType)
	sev, _ := url.QueryUnescape(severity)
	intID, _ := url.QueryUnescape(integrationID)

	switch r.Method {
	case http.MethodPost:
		for _, route := range re.Routing {
			if route.ID == intID && strings.EqualFold(route.Severity, sev) {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		re.Routing = append(re.Routing, axonopsClient.IntegrationRoute{ID: intID, Severity: sev})
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		var kept []axonopsClient.IntegrationRoute
		for _, route := range re.Routing {
			if route.ID == intID && strings.EqualFold(route.Severity, sev) {
				continue
			}
			kept = append(kept, route)
		}
		re.Routing = kept
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// --- silence windows ---

func (m *mockAxonOpsServer) handleSilence(w http.ResponseWriter, r *http.Request, clusterType, clusterName string, rest []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := clusterKey(clusterType, clusterName)

	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			list := m.silences[key]
			if list == nil {
				list = []axonopsClient.SilenceWindow{}
			}
			writeJSON(w, http.StatusOK, list)
			return
		case http.MethodPost:
			var s axonopsClient.SilenceWindow
			if err := json.Unmarshal(readBody(r), &s); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			// The real API assigns its own server-side ID, ignoring any
			// client-supplied value.
			s.ID = uuid.New().String()
			m.silences[key] = append(m.silences[key], s)
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodDelete {
		var ids []string
		_ = json.Unmarshal(readBody(r), &ids)
		idSet := map[string]bool{}
		for _, id := range ids {
			idSet[id] = true
		}
		// Fall back to the path segment if no body was sent.
		if len(idSet) == 0 {
			idSet[rest[0]] = true
		}
		var kept []axonopsClient.SilenceWindow
		for _, s := range m.silences[key] {
			if !idSet[s.ID] {
				kept = append(kept, s)
			}
		}
		m.silences[key] = kept
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}
