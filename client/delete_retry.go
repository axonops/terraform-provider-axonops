package axonopsClient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// deleteMaxRetries is the maximum number of additional attempts beyond the
// initial request when a DELETE encounters a retryable failure. Total maximum
// requests sent is therefore deleteMaxRetries + 1.
const deleteMaxRetries = 3

// deleteRetryBaseBackoff is the base for exponential backoff between retries
// (1s, 2s, 4s for the default of three retries). Declared as a var so unit
// tests can override it to keep test wall-clock time short.
var deleteRetryBaseBackoff = 1 * time.Second

// isRetryableDeleteStatus reports whether a DELETE response status warrants a
// retry. Transient gateway errors from upstream proxies (Cloudflare, ALB)
// often surface as 502/503/504 even when the backend itself is healthy.
func isRetryableDeleteStatus(code int) bool {
	return code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// isDeleteSuccess reports whether a DELETE response status indicates the
// resource is gone (or never existed). 404 is treated as success because a
// retry that reaches the API after the original request already deleted the
// resource will see the resource missing — that is the correct idempotent
// outcome, not a failure.
func isDeleteSuccess(code int) bool {
	return code == http.StatusOK ||
		code == http.StatusNoContent ||
		code == http.StatusNotFound
}

// doDeleteWithRetry sends a DELETE request and transparently retries on
// transport errors and on 502/503/504 responses, with exponential backoff. If
// the request carries a body, the caller must pass the original bytes so the
// helper can rebuild a fresh reader on every attempt — http.Request.Body is a
// single-use io.ReadCloser. 4xx responses other than 404 are returned
// immediately without retry: they are definitive API rejections. 404 is
// returned without retry because it is treated as success by the caller.
//
// Any context attached to req via http.NewRequestWithContext is honoured: a
// cancelled context aborts the retry loop immediately.
func (c *AxonopsHttpClient) doDeleteWithRetry(req *http.Request, body []byte) (*http.Response, error) {
	ctx := req.Context()
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= deleteMaxRetries; attempt++ {
		if body != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			req.ContentLength = int64(len(body))
		}

		resp, err := c.client.Do(req)
		lastResp, lastErr = resp, err

		if err == nil && !isRetryableDeleteStatus(resp.StatusCode) {
			return resp, nil
		}

		// We are going to retry (or give up). Release the previous response body.
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if attempt == deleteMaxRetries {
			break
		}

		sleep := deleteRetryBaseBackoff << attempt
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	attempts := deleteMaxRetries + 1
	if lastErr != nil {
		return nil, fmt.Errorf("DELETE failed after %d attempts: %w", attempts, lastErr)
	}
	return lastResp, fmt.Errorf("DELETE returned status %d after %d attempts", lastResp.StatusCode, attempts)
}
