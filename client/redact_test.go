package axonopsClient

import (
	"strings"
	"testing"
)

func TestRedactBody_MasksSensitiveKeys(t *testing.T) {
	body := []byte(`{"name":"c1","config":{"database.password":"hunter2"},"params":{"url":"https://hooks.slack.com/x"},"remoteConfig":"key=secret","nested":[{"apiKey":"abc"}],"partitions":3}`)
	got := redactBody(body)

	for _, secret := range []string{"hunter2", "hooks.slack.com", "key=secret", "abc"} {
		if strings.Contains(got, secret) {
			t.Errorf("redacted body still contains %q: %s", secret, got)
		}
	}
	for _, keep := range []string{`"name":"c1"`, `"partitions":3`} {
		if !strings.Contains(got, keep) {
			t.Errorf("redacted body lost non-sensitive field %s: %s", keep, got)
		}
	}
}

func TestRedactBody_NonJSONHidden(t *testing.T) {
	got := redactBody([]byte("password=hunter2"))
	if strings.Contains(got, "hunter2") {
		t.Errorf("non-JSON body leaked: %s", got)
	}
}
