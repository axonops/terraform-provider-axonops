package axonopsClient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type KafkaBrokerInfo struct {
	BrokerID   int32               `json:"brokerId"`
	LogDirSize int64               `json:"logDirSize"`
	Address    string              `json:"address"`
	Rack       *string             `json:"rack"`
	Configs    []BrokerConfigEntry `json:"configs"`
}

type BrokerConfigEntry struct {
	Name            string  `json:"name"`
	Value           *string `json:"value"`
	Source          string  `json:"source"`
	Type            string  `json:"type"`
	IsExplicitlySet bool    `json:"isExplicitlySet"`
	IsDefaultValue  bool    `json:"isDefaultValue"`
	IsReadOnly      bool    `json:"isReadOnly"`
	IsSensitive     bool    `json:"isSensitive"`
}

// GetKafkaBroker returns a broker and its configuration. When configNames is
// not empty only those configs are returned. The AxonOps API has no endpoint
// to alter broker configs.
func (c *AxonopsHttpClient) GetKafkaBroker(ctx context.Context, clusterName string, brokerID int64, configNames []string) (*KafkaBrokerInfo, error) {
	reqURL := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/broker/%d", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid), esc(clusterName), brokerID)
	if len(configNames) > 0 {
		reqURL += "?configNames=" + url.QueryEscape(strings.Join(configNames, ","))
	}
	status, body, err := c.doJSON(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to get kafka broker %d: status %d for url %v, body: %s", brokerID, status, reqURL, redactBody(body))
	}
	var info KafkaBrokerInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to decode kafka broker response: %w", err)
	}
	return &info, nil
}
