package axonopsClient

import (
	"context"
	"encoding/json"
	"fmt"
)

// ValidCompatibilityLevels lists the Schema Registry compatibility levels.
var ValidCompatibilityLevels = []string{
	"NONE",
	"BACKWARD",
	"BACKWARD_TRANSITIVE",
	"FORWARD",
	"FORWARD_TRANSITIVE",
	"FULL",
	"FULL_TRANSITIVE",
}

// schemaRegistryConfigURL returns the global config URL when subject is
// empty, else the subject-level config URL.
func (c *AxonopsHttpClient) schemaRegistryConfigURL(clusterName, subject string) string {
	u := fmt.Sprintf("%s://%s/%s/%s/kafka/%s/registry/configs", c.protocol, c.axonopsHost, axonops_api_version, esc(c.orgid), esc(clusterName))
	if subject != "" {
		u += "/" + esc(subject)
	}
	return u
}

// GetSchemaCompatibility returns the compatibility level set on subject, or
// the global level when subject is empty. It returns "" when the subject has
// no subject-level setting (it inherits the global level).
func (c *AxonopsHttpClient) GetSchemaCompatibility(ctx context.Context, clusterName, subject string) (string, error) {
	reqURL := c.schemaRegistryConfigURL(clusterName, subject)
	status, body, err := c.doJSON(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	// The subject GET answers 201 on success.
	switch status {
	case 200, 201:
	case 404:
		return "", nil
	default:
		return "", fmt.Errorf("failed to get schema registry config: status %d for url %v, body: %s", status, reqURL, redactBody(body))
	}

	// A subject without its own setting returns null.
	var cfg map[string]interface{}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return "", fmt.Errorf("failed to decode schema registry config: %w", err)
	}
	for _, key := range []string{"compatibilityLevel", "compatibility"} {
		if v, ok := cfg[key].(string); ok {
			return v, nil
		}
	}
	return "", nil
}

// SetSchemaCompatibility sets the compatibility level on subject, or the
// global level when subject is empty.
func (c *AxonopsHttpClient) SetSchemaCompatibility(ctx context.Context, clusterName, subject, level string) error {
	reqURL := c.schemaRegistryConfigURL(clusterName, subject)
	status, body, err := c.doJSON(ctx, "PUT", reqURL, map[string]string{"compatibility": level})
	if err != nil {
		return err
	}
	if status < 200 || status > 299 {
		return fmt.Errorf("failed to set schema registry config: status %d for url %v, body: %s", status, reqURL, redactBody(body))
	}
	return nil
}
