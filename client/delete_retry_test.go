package axonopsClient

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// withFastBackoff swaps the package-level retry backoff to a near-zero value
// for the duration of a test. The original value is restored on cleanup.
func withFastBackoff(t *testing.T) {
	t.Helper()
	prev := deleteRetryBaseBackoff
	deleteRetryBaseBackoff = 1 * time.Millisecond
	t.Cleanup(func() { deleteRetryBaseBackoff = prev })
}

func newTestClient(t *testing.T, server *httptest.Server) *AxonopsHttpClient {
	t.Helper()
	return &AxonopsHttpClient{
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
		protocol:    "http",
		axonopsHost: strings.TrimPrefix(server.URL, "http://"),
		apiKey:      "test",
		orgid:       "org",
		tokenType:   "Bearer",
	}
}

func newRequest(t *testing.T, ctx context.Context, server *httptest.Server) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}

func TestDoDeleteWithRetry_502ThenSuccess_ReturnsNil(t *testing.T) {
	withFastBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	resp, err := c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("want 3 server calls, got %d", got)
	}
}

func TestDoDeleteWithRetry_AllAttempts502_ReturnsError(t *testing.T) {
	withFastBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error should mention status 502, got %q", err)
	}
	if !strings.Contains(err.Error(), "4 attempts") {
		t.Errorf("error should mention attempt count, got %q", err)
	}
	if got := atomic.LoadInt32(&calls); got != deleteMaxRetries+1 {
		t.Fatalf("want %d calls, got %d", deleteMaxRetries+1, got)
	}
}

func TestDoDeleteWithRetry_503And504_Retry(t *testing.T) {
	for _, code := range []int{http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			withFastBackoff(t)
			var calls int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if atomic.AddInt32(&calls, 1) < 2 {
					w.WriteHeader(code)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			c := newTestClient(t, server)
			resp, err := c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("want 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestDoDeleteWithRetry_404_NoRetry(t *testing.T) {
	withFastBackoff(t)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	resp, err := c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("want exactly 1 call, got %d", got)
	}
}

func TestDoDeleteWithRetry_4xx_NoRetry(t *testing.T) {
	for _, code := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusUnauthorized} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			withFastBackoff(t)
			var calls int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				w.WriteHeader(code)
			}))
			defer server.Close()
			c := newTestClient(t, server)
			resp, err := c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != code {
				t.Fatalf("want %d, got %d", code, resp.StatusCode)
			}
			if got := atomic.LoadInt32(&calls); got != 1 {
				t.Fatalf("want exactly 1 call, got %d", got)
			}
		})
	}
}

func TestDoDeleteWithRetry_TransportError_Retries(t *testing.T) {
	withFastBackoff(t)
	c := &AxonopsHttpClient{
		client:    &http.Client{Timeout: 100 * time.Millisecond},
		orgid:     "org",
		tokenType: "Bearer",
	}
	// http://127.0.0.1:1 is reliably refused; the std-lib returns a transport
	// error rather than a response.
	req, err := http.NewRequest(http.MethodDelete, "http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.doDeleteWithRetry(req, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "4 attempts") {
		t.Errorf("error should mention attempt count, got %q", err)
	}
}

func TestDoDeleteWithRetry_ContextCancelled_StopsRetrying(t *testing.T) {
	prev := deleteRetryBaseBackoff
	deleteRetryBaseBackoff = 100 * time.Millisecond
	t.Cleanup(func() { deleteRetryBaseBackoff = prev })

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	c := newTestClient(t, server)
	_, err := c.doDeleteWithRetry(newRequest(t, ctx, server), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got > 2 {
		t.Fatalf("expected to bail out before second retry, got %d calls", got)
	}
}

func TestDoDeleteWithRetry_BodyResentOnEveryAttempt(t *testing.T) {
	withFastBackoff(t)
	const wantBody = `{"hello":"world"}`
	var bodies []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		if len(bodies) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	req, err := http.NewRequest(http.MethodDelete, server.URL, strings.NewReader(wantBody))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.doDeleteWithRetry(req, []byte(wantBody)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 3 {
		t.Fatalf("want 3 attempts, got %d", len(bodies))
	}
	for i, b := range bodies {
		if b != wantBody {
			t.Errorf("attempt %d body = %q, want %q", i, b, wantBody)
		}
	}
}

func TestDoDeleteWithRetry_BackoffTotalSleep(t *testing.T) {
	prev := deleteRetryBaseBackoff
	deleteRetryBaseBackoff = 50 * time.Millisecond
	t.Cleanup(func() { deleteRetryBaseBackoff = prev })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	start := time.Now()
	_, _ = c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
	elapsed := time.Since(start)

	// Three sleeps: 50ms + 100ms + 200ms = 350ms minimum.
	const wantMin = 350 * time.Millisecond
	if elapsed < wantMin {
		t.Fatalf("expected at least %v of backoff, got %v", wantMin, elapsed)
	}
}

func TestDoDeleteWithRetry_ConcurrentCalls_Race(t *testing.T) {
	withFastBackoff(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.doDeleteWithRetry(newRequest(t, context.Background(), server), nil)
		}()
	}
	wg.Wait()
}
