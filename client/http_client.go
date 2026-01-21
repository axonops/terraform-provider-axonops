package axonopsClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var axonops_api_version = "api/v1"

type AxonopsHttpClient struct {
	client      *http.Client
	protocol    string
	axonopsHost string
	apiKey      string
	orgid       string
	tokenType   string
}

func CreateHTTPClient(protocol, axonopsHost, apiKey, orgid, tokenType string) *AxonopsHttpClient {

	return &AxonopsHttpClient{
		protocol:    protocol,
		axonopsHost: axonopsHost,
		apiKey:      apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		orgid:     orgid,
		tokenType: tokenType,
	}
}

// {
// 	"configs": [
// 	  {
// 		"name": "string",
// 		"value": "string"
// 	  }
// 	],
// 	"partitionCount": 0,
// 	"replicationFactor": 0,
// 	"topicName": "string"
// }

type KafkaTopic struct {
	TopicName         string             `json:"topicName"`
	PartitionCount    int32              `json:"partitionCount"`
	ReplicationFactor int32              `json:"replicationFactor"`
	Configs           []KafkaTopicConfig `json:"configs"`
}

type KafkaTopicConfig struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *AxonopsHttpClient) CreateTopic(topicName, clusterName string, partitionCount, replicationFactor int32, topicConfigs []KafkaTopicConfig) error {

	payload := KafkaTopic{
		TopicName:         topicName,
		PartitionCount:    partitionCount,
		ReplicationFactor: replicationFactor,
		Configs:           topicConfigs,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/topics", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		return nil
	} else {
		return fmt.Errorf("failed to send POST request: status %d for url %v with topicName:%v", resp.StatusCode, url, topicName)
	}

}

func (c *AxonopsHttpClient) DeleteTopic(topicName, clusterName string) error {

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/topics/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, topicName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to send DELETE request: status %d for url %v with topicName:%v", resp.StatusCode, url, topicName)
	}
}

type ConfigsWrapper struct {
	Configs []KafkaUpdateTopicConfig `json:"configs"`
}

type KafkaUpdateTopicConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Op    string `json:"op"`
}

func (c *AxonopsHttpClient) UpdateTopicConfig(topicName, clusterName string, partitionCount, replicationFactor int32, topicConfigs []KafkaUpdateTopicConfig) error {

	payload := ConfigsWrapper{
		Configs: topicConfigs,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/topics/%s/configs", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, topicName)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to send PUT request: status %d for url %v with topicName:%v and payload %+v", resp.StatusCode, url, topicName, payload)
	}
}

// ACL types and methods

type KafkaACL struct {
	ResourceType        string `json:"resourceType"`
	ResourceName        string `json:"resourceName"`
	ResourcePatternType string `json:"resourcePatternType"`
	Principal           string `json:"principal"`
	Host                string `json:"host"`
	Operation           string `json:"operation"`
	PermissionType      string `json:"permissionType"`
}

func (c *AxonopsHttpClient) CreateACL(clusterName string, acl KafkaACL) error {
	payloadJson, err := json.Marshal(acl)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/acls", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to create ACL: status %d for url %v with acl:%+v", resp.StatusCode, url, acl)
	}
}

func (c *AxonopsHttpClient) DeleteACL(clusterName string, acl KafkaACL) error {
	payloadJson, err := json.Marshal(acl)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/acls", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName)

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to delete ACL: status %d for url %v with acl:%+v", resp.StatusCode, url, acl)
	}
}

// Kafka Connect Connector types and methods

type KafkaConnector struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

type KafkaConnectorConfig struct {
	Config map[string]string `json:"config"`
}

type KafkaConnectorResponse struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
	Tasks  []ConnectorTask   `json:"tasks"`
	Type   string            `json:"type"`
}

type ConnectorTask struct {
	Connector string `json:"connector"`
	Task      int    `json:"task"`
}

// ConnectorsListResponse represents the response from the connectors list endpoint
type ConnectorsListResponse struct {
	ClusterName    string                          `json:"clusterName"`
	ClusterAddress string                          `json:"clusterAddress"`
	Connectors     map[string]ConnectorListEntry   `json:"connectors"`
}

type ConnectorListEntry struct {
	Info   KafkaConnectorResponse `json:"info"`
	Status ConnectorStatus        `json:"status"`
}

type ConnectorStatus struct {
	Name      string                 `json:"name"`
	Connector ConnectorStateInfo     `json:"connector"`
	Tasks     []ConnectorTaskStatus  `json:"tasks"`
	Type      string                 `json:"type"`
}

type ConnectorStateInfo struct {
	State    string `json:"state"`
	WorkerId string `json:"worker_id"`
}

type ConnectorTaskStatus struct {
	Id       int    `json:"id"`
	State    string `json:"state"`
	WorkerId string `json:"worker_id"`
	Trace    string `json:"trace"`
}

func (c *AxonopsHttpClient) CreateConnector(clusterName, connectClusterName string, connector KafkaConnector) (*KafkaConnectorResponse, error) {
	payloadJson, err := json.Marshal(connector)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/connect/%s/connector", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, connectClusterName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		var result KafkaConnectorResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to create connector: status %d for url %v with connector:%+v", resp.StatusCode, url, connector)
	}
}

func (c *AxonopsHttpClient) GetConnector(clusterName, connectClusterName, connectorName string) (*KafkaConnectorResponse, error) {
	// Use the connectors list endpoint and filter for the specific connector
	// The single connector GET endpoint has known issues with AxonOps API
	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/connect/%s/connectors", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, connectClusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result ConnectorsListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Find the specific connector in the map
		if connector, exists := result.Connectors[connectorName]; exists {
			return &connector.Info, nil
		}
		return nil, nil // Connector not found
	} else if resp.StatusCode == 404 {
		return nil, nil // Connect cluster or connector not found
	} else {
		return nil, fmt.Errorf("failed to get connector: status %d for url %v", resp.StatusCode, url)
	}
}

func (c *AxonopsHttpClient) UpdateConnectorConfig(clusterName, connectClusterName, connectorName string, config map[string]string) (*KafkaConnectorResponse, error) {
	payload := KafkaConnectorConfig{
		Config: config,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/connect/%s/%s/config", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, connectClusterName, connectorName)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create PUT request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result KafkaConnectorResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to update connector config: status %d for url %v", resp.StatusCode, url)
	}
}

func (c *AxonopsHttpClient) DeleteConnector(clusterName, connectClusterName, connectorName string) error {
	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/connect/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, connectClusterName, connectorName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("failed to delete connector: status %d for url %v", resp.StatusCode, url)
	}
}

// Schema Registry types and methods

type SchemaReference struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

type CreateSchemaRequest struct {
	Schema     string            `json:"schema"`
	SchemaType string            `json:"schemaType"`
	References []SchemaReference `json:"references,omitempty"`
}

type CreateSchemaResponse struct {
	Id int `json:"id"`
}

type SchemaRegistryVersionedSchema struct {
	Id            int               `json:"id"`
	Version       int               `json:"version"`
	Schema        string            `json:"schema"`
	Type          string            `json:"type"`
	References    []SchemaReference `json:"references"`
	IsSoftDeleted bool              `json:"isSoftDeleted"`
}

func (c *AxonopsHttpClient) CreateSchema(clusterName, subject string, schema CreateSchemaRequest) (*CreateSchemaResponse, error) {
	payloadJson, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/registry/subjects/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, subject)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		var result CreateSchemaResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to create schema: status %d for url %v", resp.StatusCode, url)
	}
}

func (c *AxonopsHttpClient) GetSchema(clusterName, subject string, version string) (*SchemaRegistryVersionedSchema, error) {
	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/registry/subjects/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, subject, version)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result SchemaRegistryVersionedSchema
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else if resp.StatusCode == 404 {
		return nil, nil // Schema not found
	} else {
		return nil, fmt.Errorf("failed to get schema: status %d for url %v", resp.StatusCode, url)
	}
}

func (c *AxonopsHttpClient) DeleteSchema(clusterName, subject string) error {
	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/registry/subjects/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, subject)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to delete schema: status %d for url %v", resp.StatusCode, url)
	}
}

// Log Collector types and methods

type LogCollectorConfig struct {
	Name               string   `json:"name"`
	UUID               string   `json:"uuid"`
	Filename           string   `json:"filename"`
	DateFormat         string   `json:"dateFormat"`
	InfoRegex          string   `json:"infoRegex"`
	WarningRegex       string   `json:"warningRegex"`
	ErrorRegex         string   `json:"errorRegex"`
	DebugRegex         string   `json:"debugRegex"`
	SupportedAgentType []string `json:"supportedAgentType"`
	ErrorAlertThreshold int     `json:"errorAlertThreshold,omitempty"`
}

func (c *AxonopsHttpClient) GetLogCollectors(clusterName string) ([]LogCollectorConfig, error) {
	url := fmt.Sprintf("%s://%s/api/v1/logcollectors/%s/kafka/%s", c.protocol, c.axonopsHost, c.orgid, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result []LogCollectorConfig
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("failed to get log collectors: status %d for url %v", resp.StatusCode, url)
	}
}

func (c *AxonopsHttpClient) UpdateLogCollectors(clusterName string, collectors []LogCollectorConfig) error {
	collectorsJson, err := json.Marshal(collectors)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	reqUrl := fmt.Sprintf("%s://%s/api/v1/logcollectors/%s/kafka/%s", c.protocol, c.axonopsHost, c.orgid, clusterName)

	// The API expects form-urlencoded data with addlogs parameter
	// URL-encode the JSON to properly handle special characters
	formData := "addlogs=" + url.QueryEscape(string(collectorsJson))

	req, err := http.NewRequest("PUT", reqUrl, bytes.NewBufferString(formData))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w for url %v", err, reqUrl)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to update log collectors: status %d for url %v", resp.StatusCode, reqUrl)
	}
}

// Healthcheck types and methods

type HealthcheckIntegrations struct {
	Type            string   `json:"Type"`
	Routing         []string `json:"Routing"`
	OverrideInfo    bool     `json:"OverrideInfo"`
	OverrideWarning bool     `json:"OverrideWarning"`
	OverrideError   bool     `json:"OverrideError"`
}

type ShellHealthcheck struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Interval     string                  `json:"interval"`
	Timeout      string                  `json:"timeout"`
	Integrations HealthcheckIntegrations `json:"integrations"`
	Readonly     bool                    `json:"readonly"`
	Shell        string                  `json:"shell"`
	Script       string                  `json:"script"`
}

type HTTPHealthcheck struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Interval           string                  `json:"interval"`
	Timeout            string                  `json:"timeout"`
	Integrations       HealthcheckIntegrations `json:"integrations"`
	Readonly           bool                    `json:"readonly"`
	SupportedAgentType []string                `json:"supportedAgentType"`
	URL                string                  `json:"url"`
	Method             string                  `json:"method"`
	Headers            map[string]string       `json:"headers,omitempty"`
	Body               string                  `json:"body,omitempty"`
	ExpectedStatus     int                     `json:"expectedStatus,omitempty"`
}

type TCPHealthcheck struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Interval           string                  `json:"interval"`
	Timeout            string                  `json:"timeout"`
	Integrations       HealthcheckIntegrations `json:"integrations"`
	Readonly           bool                    `json:"readonly"`
	SupportedAgentType []string                `json:"supportedAgentType"`
	TCP                string                  `json:"tcp"`
}

type HealthchecksResponse struct {
	ShellChecks []ShellHealthcheck `json:"shellchecks"`
	HTTPChecks  []HTTPHealthcheck  `json:"httpchecks"`
	TCPChecks   []TCPHealthcheck   `json:"tcpchecks"`
}

func (c *AxonopsHttpClient) GetHealthchecks(clusterName string) (*HealthchecksResponse, error) {
	url := fmt.Sprintf("%s://%s/api/v1/healthchecks/%s/kafka/%s", c.protocol, c.axonopsHost, c.orgid, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	// Set headers
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result HealthchecksResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to get healthchecks: status %d for url %v", resp.StatusCode, url)
	}
}

func (c *AxonopsHttpClient) UpdateHealthchecks(clusterName string, healthchecks HealthchecksResponse) error {
	payloadJson, err := json.Marshal(healthchecks)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	reqUrl := fmt.Sprintf("%s://%s/api/v1/healthchecks/%s/kafka/%s", c.protocol, c.axonopsHost, c.orgid, clusterName)

	req, err := http.NewRequest("PUT", reqUrl, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w for url %v", err, reqUrl)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to update healthchecks: status %d for url %v", resp.StatusCode, reqUrl)
	}
}
