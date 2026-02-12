package axonopsClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

var axonops_api_version = "api/v1"

// debugLog prints debug information if AXONOPS_DEBUG environment variable is set
func debugLog(format string, args ...interface{}) {
	if os.Getenv("AXONOPS_DEBUG") != "" {
		fmt.Printf("[AXONOPS DEBUG] "+format+"\n", args...)
	}
}

// debugRequest logs request details for debugging
func debugRequest(req *http.Request, body []byte) {
	if os.Getenv("AXONOPS_DEBUG") == "" {
		return
	}
	fmt.Printf("[AXONOPS DEBUG] === REQUEST ===\n")
	fmt.Printf("[AXONOPS DEBUG] Method: %s\n", req.Method)
	fmt.Printf("[AXONOPS DEBUG] URL: %s\n", req.URL.String())
	fmt.Printf("[AXONOPS DEBUG] Headers:\n")
	for key, values := range req.Header {
		for _, value := range values {
			// Mask API key for security
			if key == "Authorization" {
				if len(value) > 20 {
					fmt.Printf("[AXONOPS DEBUG]   %s: %s...%s\n", key, value[:15], value[len(value)-4:])
				} else {
					fmt.Printf("[AXONOPS DEBUG]   %s: %s\n", key, value)
				}
			} else {
				fmt.Printf("[AXONOPS DEBUG]   %s: %s\n", key, value)
			}
		}
	}
	if body != nil && len(body) > 0 {
		fmt.Printf("[AXONOPS DEBUG] Body: %s\n", string(body))
	}
}

// debugResponse logs response details for debugging
func debugResponse(resp *http.Response, body []byte) {
	if os.Getenv("AXONOPS_DEBUG") == "" {
		return
	}
	fmt.Printf("[AXONOPS DEBUG] === RESPONSE ===\n")
	fmt.Printf("[AXONOPS DEBUG] Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("[AXONOPS DEBUG] Headers:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("[AXONOPS DEBUG]   %s: %s\n", key, value)
		}
	}
	if body != nil && len(body) > 0 {
		// Truncate long responses
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			fmt.Printf("[AXONOPS DEBUG] Body (truncated): %s...\n", bodyStr[:500])
		} else {
			fmt.Printf("[AXONOPS DEBUG] Body: %s\n", bodyStr)
		}
	}
}

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

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}

	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 201 {
		return nil
	} else {
		return fmt.Errorf("failed to send POST request: status %d for url %v with topicName:%v, body: %s", resp.StatusCode, url, topicName, string(bodyBytes))
	}

}

// TopicInfo represents topic information returned from the API
type TopicInfo struct {
	Name              string             `json:"name"`
	Partitions        int32              `json:"partitionCount"`
	ReplicationFactor int32              `json:"replicationFactor"`
	Config            []KafkaTopicConfig `json:"-"` // Populated from configs endpoint
}

// TopicConfigEntry represents a config entry from the configs endpoint
type TopicConfigEntry struct {
	Name            string `json:"name"`
	Value           string `json:"value"`
	Source          string `json:"source"`
	IsExplicitlySet bool   `json:"isExplicitlySet"`
}

// TopicConfigDescription wraps config entries for a topic
type TopicConfigDescription struct {
	TopicName     string             `json:"topicName"`
	ConfigEntries []TopicConfigEntry `json:"configEntries"`
}

// TopicConfigResponse is the response from the configs endpoint
type TopicConfigResponse struct {
	TopicDescription []TopicConfigDescription `json:"topicDescription"`
}

// GetTopic retrieves a topic's information including configs
func (c *AxonopsHttpClient) GetTopic(topicName, clusterName string) (*TopicInfo, error) {
	// Get basic topic info
	topicUrl := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/topics/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, topicName)

	req, err := http.NewRequest("GET", topicUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, topicUrl)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get topic: status %d for url %v, body: %s", resp.StatusCode, topicUrl, string(bodyBytes))
	}

	var topicInfo TopicInfo
	if err := json.Unmarshal(bodyBytes, &topicInfo); err != nil {
		return nil, fmt.Errorf("failed to decode topic response: %w", err)
	}

	// Get topic configs
	configUrl := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/topics/%s/configs", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName, topicName)

	configReq, err := http.NewRequest("GET", configUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for configs: %w", err)
	}

	if c.apiKey != "" {
		configReq.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	configResp, err := c.client.Do(configReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request for configs: %w", err)
	}
	defer configResp.Body.Close()

	if configResp.StatusCode == 200 {
		var configResponse TopicConfigResponse
		if err := json.NewDecoder(configResp.Body).Decode(&configResponse); err != nil {
			return nil, fmt.Errorf("failed to decode configs response: %w", err)
		}

		// Only include explicitly set configs
		if len(configResponse.TopicDescription) > 0 {
			for _, entry := range configResponse.TopicDescription[0].ConfigEntries {
				if entry.IsExplicitlySet {
					topicInfo.Config = append(topicInfo.Config, KafkaTopicConfig{
						Name:  entry.Name,
						Value: entry.Value,
					})
				}
			}
		}
	}

	return &topicInfo, nil
}

// GetTopics retrieves all topics for a cluster
func (c *AxonopsHttpClient) GetTopics(clusterName string) ([]TopicInfo, error) {
	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/topics", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get topics: status %d for url %v", resp.StatusCode, url)
	}

	var topics []TopicInfo
	if err := json.NewDecoder(resp.Body).Decode(&topics); err != nil {
		return nil, fmt.Errorf("failed to decode topics response: %w", err)
	}

	return topics, nil
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

type ACLResource struct {
	ResourceType        string     `json:"resourceType"`
	ResourceName        string     `json:"resourceName"`
	ResourcePatternType string     `json:"resourcePatternType"`
	ACLs                []KafkaACL `json:"acls"`
}

type ACLResponse struct {
	ACLResources []ACLResource `json:"aclResources"`
}

func (c *AxonopsHttpClient) GetACLs(clusterName string) (*ACLResponse, error) {
	url := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/acls", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	debugResponse(resp, body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get ACLs: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result ACLResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode ACL response: %w", err)
	}

	return &result, nil
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

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		var result KafkaConnectorResponse
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to create connector: status %d for url %v with connector:%+v, body: %s", resp.StatusCode, url, connector, string(bodyBytes))
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

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 {
		var result ConnectorsListResponse
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
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
		return nil, fmt.Errorf("failed to get connector: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
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

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 {
		var result KafkaConnectorResponse
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to update connector config: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
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

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("failed to delete connector: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
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

// Adaptive Repair types and methods

type AdaptiveRepairSettings struct {
	Active              bool     `json:"Active"`
	GcGraceThreshold    int      `json:"GcGraceThreshold"`
	TableParallelism    int      `json:"TableParallelism"`
	BlacklistedTables   []string `json:"BlacklistedTables"`
	FilterTWCSTables    bool     `json:"FilterTWCSTables"`
	SegmentRetries      int      `json:"SegmentRetries"`
	SegmentsPerVnode    int      `json:"SegmentsPerVnode,omitempty"`
	SegmentTargetSizeMB int      `json:"SegmentTargetSizeMB,omitempty"`
}

func (c *AxonopsHttpClient) GetCassandraAdaptiveRepair(clusterType, clusterName string) (*AdaptiveRepairSettings, error) {
	url := fmt.Sprintf("%s://%s/%s/adaptiveRepair/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 {
		var result AdaptiveRepairSettings
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to get adaptive repair settings: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

func (c *AxonopsHttpClient) UpdateCassandraAdaptiveRepair(clusterType, clusterName string, settings AdaptiveRepairSettings) error {
	payloadJson, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/adaptiveRepair/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to update adaptive repair settings: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

// Cassandra Backup types and methods

type CassandraBackup struct {
	ID                      string   `json:"ID"`
	Tag                     string   `json:"tag"`
	LocalRetentionDuration  string   `json:"LocalRetentionDuration"`
	Remote                  bool     `json:"Remote"`
	RemoteConfig            string   `json:"remoteConfig,omitempty"`
	RemotePath              string   `json:"remotePath,omitempty"`
	RemoteRetentionDuration string   `json:"RemoteRetentionDuration,omitempty"`
	RemoteType              string   `json:"remoteType,omitempty"`
	Timeout                 string   `json:"timeout,omitempty"`
	Transfers               int      `json:"transfers,omitempty"`
	TpsLimit                int      `json:"tpslimit,omitempty"`
	BwLimit                 string   `json:"bwlimit,omitempty"`
	Datacenters             []string `json:"datacenters"`
	Nodes                   []string `json:"nodes"`
	Tables                  []string `json:"tables"`
	Keyspaces               []string `json:"keyspaces"`
	AllTables               bool     `json:"allTables"`
	AllNodes                bool     `json:"allNodes"`
	Schedule                bool     `json:"schedule"`
	ScheduleExpr            string   `json:"scheduleExpr"`
}

type CassandraBackupsResponse struct {
	ScheduledSnapshots []CassandraScheduledSnapshot `json:"ScheduledSnapshots"`
}

type CassandraScheduledSnapshot struct {
	ID     string          `json:"ID"`
	Params json.RawMessage `json:"Params"`
}

type CassandraScheduledParam struct {
	BackupDetails string `json:"BackupDetails"`
}

func (c *AxonopsHttpClient) GetCassandraBackups(clusterType, clusterName string) ([]CassandraBackup, error) {
	url := fmt.Sprintf("%s://%s/%s/cassandraScheduleSnapshot/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get cassandra backups: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}

	var response CassandraBackupsResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var backups []CassandraBackup
	for _, snapshot := range response.ScheduledSnapshots {
		if len(snapshot.Params) == 0 {
			continue
		}

		// Params can be a JSON string or an array of objects
		var params []CassandraScheduledParam
		if err := json.Unmarshal(snapshot.Params, &params); err != nil {
			// Try as a JSON string containing the array
			var paramsStr string
			if err2 := json.Unmarshal(snapshot.Params, &paramsStr); err2 == nil {
				json.Unmarshal([]byte(paramsStr), &params)
			}
		}

		for _, param := range params {
			if param.BackupDetails != "" {
				var backup CassandraBackup
				if err := json.Unmarshal([]byte(param.BackupDetails), &backup); err != nil {
					continue
				}
				if backup.ID == "" {
					backup.ID = snapshot.ID
				}
				backups = append(backups, backup)
			}
		}
	}

	return backups, nil
}

func (c *AxonopsHttpClient) CreateCassandraBackup(clusterType, clusterName string, backup CassandraBackup) error {
	payloadJson, err := json.Marshal(backup)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/cassandraSnapshot/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to create cassandra backup: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

func (c *AxonopsHttpClient) DeleteCassandraBackup(clusterType, clusterName string, backupIDs []string) error {
	payloadJson, err := json.Marshal(backupIDs)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/cassandraScheduleSnapshot/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("failed to delete cassandra backup: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

// Metric Alert Rule types and methods

type MetricAlertRule struct {
	ID            string                 `json:"id"`
	Alert         string                 `json:"alert"`
	For           string                 `json:"for"`
	Operator      string                 `json:"operator"`
	WarningValue  float64                `json:"warningValue"`
	CriticalValue float64                `json:"criticalValue"`
	Expr          string                 `json:"expr"`
	WidgetTitle   string                 `json:"widgetTitle,omitempty"`
	CorrelationId string                 `json:"correlationId,omitempty"`
	Annotations   MetricAlertAnnotations `json:"annotations"`
	Filters       []MetricAlertFilter    `json:"filters,omitempty"`
}

type MetricAlertAnnotations struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
	WidgetUrl   string `json:"widget_url,omitempty"`
}

type MetricAlertFilter struct {
	Name  string   `json:"Name"`
	Value []string `json:"Value"`
}

type AlertRulesResponse struct {
	MetricRules []MetricAlertRule `json:"metricrules"`
}

func (c *AxonopsHttpClient) GetAlertRules(clusterType, clusterName string) ([]MetricAlertRule, error) {
	url := fmt.Sprintf("%s://%s/%s/alert-rules/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get alert rules: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}

	var response AlertRulesResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.MetricRules, nil
}

func (c *AxonopsHttpClient) CreateOrUpdateAlertRule(clusterType, clusterName string, rule MetricAlertRule) error {
	payloadJson, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/alert-rules/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		return nil
	} else {
		return fmt.Errorf("failed to create/update alert rule: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

func (c *AxonopsHttpClient) DeleteAlertRule(clusterType, clusterName, alertID string) error {
	url := fmt.Sprintf("%s://%s/%s/alert-rules/%s/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName, alertID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("failed to delete alert rule: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

// Alert Route (Integration Routing) types and methods

type IntegrationsResponse struct {
	Definitions []IntegrationDefinition `json:"Definitions"`
	Routings    []IntegrationRouting    `json:"Routings"`
}

type IntegrationDefinition struct {
	ID     string            `json:"ID"`
	Type   string            `json:"Type"`
	Params map[string]string `json:"Params"`
}

type IntegrationRouting struct {
	Type            string             `json:"Type"`
	Routing         []IntegrationRoute `json:"Routing"`
	OverrideInfo    bool               `json:"OverrideInfo"`
	OverrideWarning bool               `json:"OverrideWarning"`
	OverrideError   bool               `json:"OverrideError"`
}

type IntegrationRoute struct {
	ID       string `json:"ID"`
	Severity string `json:"Severity"`
}

type OverridePayload struct {
	Value bool `json:"value"`
}

func (c *AxonopsHttpClient) GetIntegrations(clusterType, clusterName string) (*IntegrationsResponse, error) {
	url := fmt.Sprintf("%s://%s/%s/integrations/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 {
		var result IntegrationsResponse
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &result, nil
	} else {
		return nil, fmt.Errorf("failed to get integrations: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

func (c *AxonopsHttpClient) SetIntegrationOverride(clusterType, clusterName, routeType, severity string, value bool) error {
	payload := OverridePayload{Value: value}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	url := fmt.Sprintf("%s://%s/%s/integrations-override/%s/%s/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName, routeType, severity)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w for url %v", err, url)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, payloadJson)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("failed to set integration override: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

func (c *AxonopsHttpClient) AddIntegrationRoute(clusterType, clusterName, routeType, severity, integrationID string) error {
	url := fmt.Sprintf("%s://%s/%s/integrations-routing/%s/%s/%s/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName, routeType, severity, integrationID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 204 {
		return nil
	} else {
		return fmt.Errorf("failed to add integration route: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}

func (c *AxonopsHttpClient) RemoveIntegrationRoute(clusterType, clusterName, routeType, severity, integrationID string) error {
	url := fmt.Sprintf("%s://%s/%s/integrations-routing/%s/%s/%s/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, c.orgid, clusterType, clusterName, routeType, severity, integrationID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w for url %v", err, url)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, nil)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	debugResponse(resp, bodyBytes)

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("failed to remove integration route: status %d for url %v, body: %s", resp.StatusCode, url, string(bodyBytes))
	}
}
