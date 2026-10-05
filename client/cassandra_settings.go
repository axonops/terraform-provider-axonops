package axonopsClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Cassandra commitlog archive settings types and methods

// CommitLogArchiveSettings is one commitlog archive configuration. The API
// keeps a list per cluster with one entry per datacenter; Datacenters must
// hold exactly that one datacenter.
// The API returns RemoteRetentionDuration and RemoteConfig capitalised but
// accepts them lower-camel-case; encoding/json matches keys case-insensitively
// so a single struct serves both directions.
type CommitLogArchiveSettings struct {
	Datacenters             []string `json:"datacenters"`
	RemoteType              string   `json:"remoteType"`
	RemotePath              string   `json:"remotePath"`
	RemoteRetentionDuration string   `json:"remoteRetentionDuration"`
	RemoteConfig            string   `json:"remoteConfig"`
	Timeout                 string   `json:"timeout"`
	BwLimit                 string   `json:"bwlimit"`
	Transfers               int      `json:"transfers"`
}

// errCommitLogPITRDisabled explains the bare 400 the AxonOps server returns
// from every commitlog settings endpoint when the organisation does not have
// the Cassandra point-in-time restore (PITR) feature.
const errCommitLogPITRDisabled = "the AxonOps server rejected the request without a reason (HTTP 400); commitlog archiving requires the Cassandra point-in-time restore (PITR) feature to be enabled for the organisation"

// commitLogError builds the error for a failed commitlog settings request.
func commitLogError(action string, status int, reqURL string, body []byte) error {
	if status == 400 && len(bytes.TrimSpace(body)) == 0 {
		return fmt.Errorf("failed to %s commitlog archive settings: %s (url %v)", action, errCommitLogPITRDisabled, reqURL)
	}
	return fmt.Errorf("failed to %s commitlog archive settings: status %d for url %v, body: %s", action, status, reqURL, string(body))
}

func (c *AxonopsHttpClient) commitLogSettingsURL(clusterType, clusterName string) string {
	return fmt.Sprintf("%s://%s/%s/cassandraCommitLogsSettings/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid), esc(clusterType), esc(clusterName))
}

// GetCommitLogArchiveSettings returns every commitlog archive configuration
// of a cluster.
func (c *AxonopsHttpClient) GetCommitLogArchiveSettings(ctx context.Context, clusterType, clusterName string) ([]CommitLogArchiveSettings, error) {
	reqURL := c.commitLogSettingsURL(clusterType, clusterName)
	status, body, err := c.doJSON(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, commitLogError("get", status, reqURL, body)
	}
	var settings []CommitLogArchiveSettings
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, fmt.Errorf("failed to decode commitlog archive settings response: %w", err)
	}
	return settings, nil
}

// CreateCommitLogArchiveSettings adds a commitlog archive configuration.
func (c *AxonopsHttpClient) CreateCommitLogArchiveSettings(ctx context.Context, clusterType, clusterName string, settings CommitLogArchiveSettings) error {
	reqURL := c.commitLogSettingsURL(clusterType, clusterName)
	status, body, err := c.doJSON(ctx, "POST", reqURL, settings)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 && status != 204 {
		return commitLogError("create", status, reqURL, body)
	}
	return nil
}

// UpdateCommitLogArchiveSettings replaces the commitlog archive configuration
// of a datacenter. The API takes exactly one datacenter per configuration.
func (c *AxonopsHttpClient) UpdateCommitLogArchiveSettings(ctx context.Context, clusterType, clusterName string, settings CommitLogArchiveSettings) error {
	if len(settings.Datacenters) == 0 {
		return fmt.Errorf("commitlog archive settings must list at least one datacenter")
	}
	reqURL := c.commitLogSettingsURL(clusterType, clusterName) + "/" + esc(settings.Datacenters[0])
	status, body, err := c.doJSON(ctx, "PUT", reqURL, settings)
	if err != nil {
		return err
	}
	if status != 200 && status != 204 {
		return commitLogError("update", status, reqURL, body)
	}
	return nil
}

// DeleteCommitLogArchiveSettings removes the commitlog archive configurations
// covering the given datacenters.
func (c *AxonopsHttpClient) DeleteCommitLogArchiveSettings(ctx context.Context, clusterType, clusterName string, datacenters []string) error {
	payloadJson, err := json.Marshal(datacenters)
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	reqURL := c.commitLogSettingsURL(clusterType, clusterName)

	req, err := http.NewRequestWithContext(ctx, "DELETE", reqURL, bytes.NewBuffer(payloadJson))
	if err != nil {
		return fmt.Errorf("failed to create DELETE request for url %v: %w", reqURL, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	debugRequest(req, payloadJson)

	resp, err := c.doDeleteWithRetry(req, payloadJson)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	debugResponse(resp, bodyBytes)

	if isDeleteSuccess(resp.StatusCode) {
		return nil
	}
	return commitLogError("delete", resp.StatusCode, reqURL, bodyBytes)
}

// Agent disconnection tolerance types and methods

// AgentDisconnectionTolerance sets how long an agent may be disconnected
// before AxonOps raises a warning and then an error. Values are duration
// strings such as "30s" or "1m".
type AgentDisconnectionTolerance struct {
	WarnTimeout  string `json:"warn_timeout"`
	ErrorTimeout string `json:"error_timeout"`
}

func (c *AxonopsHttpClient) agentDisconnectionToleranceURL(clusterType, clusterName string) string {
	return fmt.Sprintf("%s://%s/%s/configs/agentDisconnectionTolerance/%s/%s/%s", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid), esc(clusterType), esc(clusterName))
}

// GetAgentDisconnectionTolerance returns the agent disconnection tolerance of
// a cluster.
func (c *AxonopsHttpClient) GetAgentDisconnectionTolerance(ctx context.Context, clusterType, clusterName string) (*AgentDisconnectionTolerance, error) {
	reqURL := c.agentDisconnectionToleranceURL(clusterType, clusterName)
	status, body, err := c.doJSON(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to get agent disconnection tolerance: status %d for url %v, body: %s", status, reqURL, string(body))
	}
	var tolerance AgentDisconnectionTolerance
	if err := json.Unmarshal(body, &tolerance); err != nil {
		return nil, fmt.Errorf("failed to decode agent disconnection tolerance response: %w", err)
	}
	return &tolerance, nil
}

// UpdateAgentDisconnectionTolerance sets the agent disconnection tolerance of
// a cluster.
func (c *AxonopsHttpClient) UpdateAgentDisconnectionTolerance(ctx context.Context, clusterType, clusterName string, tolerance AgentDisconnectionTolerance) error {
	reqURL := c.agentDisconnectionToleranceURL(clusterType, clusterName)
	status, body, err := c.doJSON(ctx, "PUT", reqURL, tolerance)
	if err != nil {
		return err
	}
	if status != 200 && status != 204 {
		return fmt.Errorf("failed to update agent disconnection tolerance: status %d for url %v, body: %s", status, reqURL, string(body))
	}
	return nil
}
