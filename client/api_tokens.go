package axonopsClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ApiToken is an AxonOps API token as listed by the server. Tokens are
// immutable: there is no update or rotate endpoint.
type ApiToken struct {
	AllowedRoles []string `json:"allowed_roles"`
	// TokenExpiry is epoch seconds; 0 means the token never expires.
	TokenExpiry  int32  `json:"token_expiry_time"`
	KeyId        string `json:"key_id,omitempty"`
	KeyHash      string `json:"key_hash,omitempty"`
	CreationTime int32  `json:"creation_time,omitempty"`
}

// CreateApiTokenResponse carries the token secret. The server returns the
// secret only here and stores just its hash.
type CreateApiTokenResponse struct {
	ApiKey   string `json:"apiKey"`
	ApiKeyId string `json:"apiKeyId"`
}

func (c *AxonopsHttpClient) CreateApiToken(ctx context.Context, allowedRoles []string, expiry int32) (*CreateApiTokenResponse, error) {
	reqURL := fmt.Sprintf("%s://%s/%s/%s/createApiToken", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid))
	status, body, err := c.doJSON(ctx, "POST", reqURL, ApiToken{AllowedRoles: allowedRoles, TokenExpiry: expiry})
	if err != nil {
		return nil, err
	}
	if status < 200 || status > 299 {
		return nil, fmt.Errorf("failed to create API token: status %d for url %v, body: %s", status, reqURL, redactBody(body))
	}
	var result CreateApiTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode create API token response: %w", err)
	}
	if result.ApiKeyId == "" || result.ApiKey == "" {
		return nil, fmt.Errorf("create API token response is missing apiKey or apiKeyId")
	}
	return &result, nil
}

// ListApiTokens returns all API tokens of the organisation.
func (c *AxonopsHttpClient) ListApiTokens(ctx context.Context) ([]ApiToken, error) {
	reqURL := fmt.Sprintf("%s://%s/%s/%s/listApiTokens", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid))
	status, body, err := c.doJSON(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("failed to list API tokens: status %d for url %v, body: %s", status, reqURL, string(body))
	}
	// An organisation without tokens returns null.
	var tokens []ApiToken
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to decode API token list: %w", err)
	}
	return tokens, nil
}

// GetApiToken returns the token with the given key ID, or nil if not found.
func (c *AxonopsHttpClient) GetApiToken(ctx context.Context, keyId string) (*ApiToken, error) {
	tokens, err := c.ListApiTokens(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		if tokens[i].KeyId == keyId {
			return &tokens[i], nil
		}
	}
	return nil, nil
}

// DeleteApiToken revokes the token with the given key ID. The server deletes
// by key hash, so the hash is looked up from the token list first. A token
// that no longer exists is treated as deleted.
func (c *AxonopsHttpClient) DeleteApiToken(ctx context.Context, keyId string) error {
	token, err := c.GetApiToken(ctx, keyId)
	if err != nil {
		return err
	}
	if token == nil {
		return nil
	}

	reqURL := fmt.Sprintf("%s://%s/%s/%s/deleteApiToken", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid))
	payload, err := json.Marshal(map[string]string{"key_hash": token.KeyHash})
	if err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE", reqURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create DELETE request for url %v: %w", reqURL, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.apiKey)
	}

	resp, err := c.doDeleteWithRetry(req, payload)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if isDeleteSuccess(resp.StatusCode) {
		return nil
	}
	// The server answers 500 for an unknown hash, so a token revoked between
	// the lookup and the DELETE also lands here: treat it as deleted.
	if gone, err := c.GetApiToken(ctx, keyId); err == nil && gone == nil {
		return nil
	}
	return fmt.Errorf("failed to delete API token %s: status %d for url %v", keyId, resp.StatusCode, reqURL)
}
