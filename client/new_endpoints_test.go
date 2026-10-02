package axonopsClient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubServer answers every request with status and body, and records the
// last request path and query.
func stubServer(t *testing.T, status int, body string) (*httptest.Server, *string) {
	t.Helper()
	var last string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = r.URL.RequestURI()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, &last
}

func TestNewEndpoints_ErrorStatus_ReturnsError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func(c *AxonopsHttpClient) error
	}{
		{"GetDashboardTemplate", func(c *AxonopsHttpClient) error {
			_, err := c.GetDashboardTemplate(ctx, "kafka", "k")
			return err
		}},
		{"SetDashboardTemplate", func(c *AxonopsHttpClient) error {
			return c.SetDashboardTemplate(ctx, "kafka", "k", DashboardTemplate{})
		}},
		{"CreateApiToken", func(c *AxonopsHttpClient) error {
			_, err := c.CreateApiToken(ctx, []string{"org/admin"}, 0)
			return err
		}},
		{"ListApiTokens", func(c *AxonopsHttpClient) error {
			_, err := c.ListApiTokens(ctx)
			return err
		}},
		{"GetSchemaCompatibility", func(c *AxonopsHttpClient) error {
			_, err := c.GetSchemaCompatibility(ctx, "k", "s")
			return err
		}},
		{"SetSchemaCompatibility", func(c *AxonopsHttpClient) error {
			return c.SetSchemaCompatibility(ctx, "k", "s", "FULL")
		}},
		{"GetKafkaBroker", func(c *AxonopsHttpClient) error {
			_, err := c.GetKafkaBroker(ctx, "k", 1, nil)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := stubServer(t, http.StatusForbidden, `{"error":"denied"}`)
			err := tt.call(newTestClient(t, server))
			if err == nil || !strings.Contains(err.Error(), "status 403") {
				t.Fatalf("want status 403 error, got %v", err)
			}
		})
	}
}

func TestNewEndpoints_InvalidJSON_ReturnsDecodeError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func(c *AxonopsHttpClient) error
	}{
		{"GetDashboardTemplate", func(c *AxonopsHttpClient) error {
			_, err := c.GetDashboardTemplate(ctx, "kafka", "k")
			return err
		}},
		{"CreateApiToken", func(c *AxonopsHttpClient) error {
			_, err := c.CreateApiToken(ctx, []string{"org/admin"}, 0)
			return err
		}},
		{"ListApiTokens", func(c *AxonopsHttpClient) error {
			_, err := c.ListApiTokens(ctx)
			return err
		}},
		{"GetSchemaCompatibility", func(c *AxonopsHttpClient) error {
			_, err := c.GetSchemaCompatibility(ctx, "k", "")
			return err
		}},
		{"GetKafkaBroker", func(c *AxonopsHttpClient) error {
			_, err := c.GetKafkaBroker(ctx, "k", 1, nil)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := stubServer(t, http.StatusOK, `not json`)
			if err := tt.call(newTestClient(t, server)); err == nil || !strings.Contains(err.Error(), "decode") {
				t.Fatalf("want decode error, got %v", err)
			}
		})
	}
}

func TestCreateApiToken_ErrorBody_RedactsSecret(t *testing.T) {
	server, _ := stubServer(t, http.StatusInternalServerError, `{"apiKey":"s3cr3t-value"}`)
	_, err := newTestClient(t, server).CreateApiToken(context.Background(), []string{"org/admin"}, 0)
	if err == nil || strings.Contains(err.Error(), "s3cr3t-value") {
		t.Fatalf("want error without secret, got %v", err)
	}
}

func TestCreateApiToken_MissingFields_ReturnsError(t *testing.T) {
	server, _ := stubServer(t, http.StatusOK, `{"apiKeyId":"id-only"}`)
	_, err := newTestClient(t, server).CreateApiToken(context.Background(), []string{"org/admin"}, 0)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("want missing-field error, got %v", err)
	}
}

func TestListApiTokens_Null_ReturnsEmpty(t *testing.T) {
	server, _ := stubServer(t, http.StatusOK, `null`)
	tokens, err := newTestClient(t, server).ListApiTokens(context.Background())
	if err != nil || len(tokens) != 0 {
		t.Fatalf("want no tokens and no error, got %v, %v", tokens, err)
	}
}

func TestDeleteApiToken_UnknownID_IsNoop(t *testing.T) {
	server, last := stubServer(t, http.StatusOK, `[{"key_id":"other","key_hash":"h"}]`)
	if err := newTestClient(t, server).DeleteApiToken(context.Background(), "missing"); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if !strings.HasSuffix(*last, "/listApiTokens") {
		t.Fatalf("want only a list call, last request was %s", *last)
	}
}

func TestDeleteApiToken_StillPresentAfterFailure_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`[{"key_id":"k1","key_hash":"h1"}]`))
	}))
	t.Cleanup(server.Close)
	err := newTestClient(t, server).DeleteApiToken(context.Background(), "k1")
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("want status 403 error, got %v", err)
	}
}

func TestGetSchemaCompatibility_Responses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"subject level answers 201", http.StatusCreated, `{"compatibilityLevel":"FULL"}`, "FULL"},
		{"echo shape", http.StatusOK, `{"compatibility":"NONE"}`, "NONE"},
		{"inherits global", http.StatusCreated, `null`, ""},
		{"not found", http.StatusNotFound, ``, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := stubServer(t, tt.status, tt.body)
			got, err := newTestClient(t, server).GetSchemaCompatibility(context.Background(), "k", "s")
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestSchemaRegistryConfigURL_EscapesSubject(t *testing.T) {
	server, last := stubServer(t, http.StatusOK, `{"compatibilityLevel":"FULL"}`)
	c := newTestClient(t, server)
	if _, err := c.GetSchemaCompatibility(context.Background(), "k", "a/b"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*last, "/registry/configs/a%2Fb") {
		t.Fatalf("subject not escaped: %s", *last)
	}
	if _, err := c.GetSchemaCompatibility(context.Background(), "k", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*last, "/registry/configs") {
		t.Fatalf("global URL wrong: %s", *last)
	}
}

func TestGetKafkaBroker_ConfigNames_SentAsQuery(t *testing.T) {
	server, last := stubServer(t, http.StatusOK, `{"brokerId":1}`)
	if _, err := newTestClient(t, server).GetKafkaBroker(context.Background(), "k", 1, []string{"a.b", "c"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*last, "/broker/1?configNames=a.b%2Cc") {
		t.Fatalf("unexpected request: %s", *last)
	}
}

func TestDashboardTemplate_UsesVersion2(t *testing.T) {
	server, last := stubServer(t, http.StatusNoContent, ``)
	if err := newTestClient(t, server).SetDashboardTemplate(context.Background(), "kafka", "k", DashboardTemplate{}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*last, "?dashver=2.0") {
		t.Fatalf("missing dashver: %s", *last)
	}
}
