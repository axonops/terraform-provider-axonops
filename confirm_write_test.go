package main

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// shortenConfirmWrite makes confirmWrite poll fast and give up after timeout,
// restoring the defaults when the test ends.
func shortenConfirmWrite(t *testing.T, timeout time.Duration) {
	t.Helper()
	oldTimeout, oldBase, oldMax := confirmWriteTimeout, confirmWriteBaseBackoff, confirmWriteMaxBackoff
	confirmWriteTimeout, confirmWriteBaseBackoff, confirmWriteMaxBackoff = timeout, time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() {
		confirmWriteTimeout, confirmWriteBaseBackoff, confirmWriteMaxBackoff = oldTimeout, oldBase, oldMax
	})
}

func TestConfirmWrite_foundOnFirstRead(t *testing.T) {
	shortenConfirmWrite(t, time.Second)
	calls := 0
	got, err := confirmWrite(context.Background(), "thing", func(context.Context) (string, bool, error) {
		calls++
		return "id-1", true, nil
	})
	if err != nil || got != "id-1" || calls != 1 {
		t.Fatalf("got (%q, %v) after %d calls; want (\"id-1\", nil) after 1", got, err, calls)
	}
}

func TestConfirmWrite_foundAfterRetries(t *testing.T) {
	shortenConfirmWrite(t, time.Second)
	calls := 0
	got, err := confirmWrite(context.Background(), "thing", func(context.Context) (string, bool, error) {
		calls++
		return "id-1", calls == 3, nil
	})
	if err != nil || got != "id-1" || calls != 3 {
		t.Fatalf("got (%q, %v) after %d calls; want (\"id-1\", nil) after 3", got, err, calls)
	}
}

func TestConfirmWrite_readErrorIsRetried(t *testing.T) {
	shortenConfirmWrite(t, time.Second)
	calls := 0
	_, err := confirmWrite(context.Background(), "thing", func(context.Context) (string, bool, error) {
		calls++
		if calls == 1 {
			return "", false, errors.New("status 502")
		}
		return "id-1", true, nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("got err %v after %d calls; want nil after 2", err, calls)
	}
}

func TestConfirmWrite_neverVisibleTimesOut(t *testing.T) {
	shortenConfirmWrite(t, 50*time.Millisecond)
	_, err := confirmWrite(context.Background(), "thing", func(context.Context) (string, bool, error) {
		return "", false, nil
	})
	if !errors.Is(err, errWriteNotConfirmed) {
		t.Fatalf("got %v; want errWriteNotConfirmed", err)
	}
}

func TestConfirmWrite_timeoutReportsLastReadError(t *testing.T) {
	shortenConfirmWrite(t, 50*time.Millisecond)
	_, err := confirmWrite(context.Background(), "thing", func(context.Context) (string, bool, error) {
		return "", false, errors.New("status 503")
	})
	if !errors.Is(err, errWriteNotConfirmed) || !regexp.MustCompile(`last read error: status 503`).MatchString(err.Error()) {
		t.Fatalf("got %v; want errWriteNotConfirmed carrying the last read error", err)
	}
}

func TestConfirmWrite_cancelledContextStops(t *testing.T) {
	shortenConfirmWrite(t, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := confirmWrite(ctx, "thing", func(context.Context) (string, bool, error) {
		calls++
		cancel()
		return "", false, nil
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("got %v after %d calls; want context.Canceled after 1", err, calls)
	}
}

const slackLagConfig = `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "lagged"
  webhook_url  = "https://hooks.slack.com/services/x"
  channel      = "#alerts"
}
`

// TestAccSlackIntegration_createWaitsForDelayedVisibility reproduces the
// "Integration was created but could not be found" failure: the API accepts
// the POST but the next list GETs do not include the new integration yet.
func TestAccSlackIntegration_createWaitsForDelayedVisibility(t *testing.T) {
	shortenConfirmWrite(t, 5*time.Second)
	srv := newAccTestServer(t)
	srv.setIntegrationReadLag(3)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccProviderConfig(srv.URL()) + slackLagConfig,
			Check:  resource.TestCheckResourceAttrSet("axonops_slack_integration.s", "id"),
		}},
	})
}

func TestAccSlackIntegration_createFailsWhenNeverVisible(t *testing.T) {
	shortenConfirmWrite(t, 200*time.Millisecond)
	srv := newAccTestServer(t)
	srv.setIntegrationReadLag(1 << 30)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      testAccProviderConfig(srv.URL()) + slackLagConfig,
			ExpectError: regexp.MustCompile(`Unable to confirm Slack integration was created`),
		}},
	})
}

func TestKeepDollarEscaped(t *testing.T) {
	cases := []struct {
		name  string
		prior types.String
		api   string
		want  string
	}{
		{"escaped spelling kept", types.StringValue("per $$groupBy"), "per $groupBy", "per $$groupBy"},
		{"plain spelling kept", types.StringValue("per $groupBy"), "per $groupBy", "per $groupBy"},
		{"renamed upstream uses API", types.StringValue("per $$groupBy"), "per $dc", "per $dc"},
		{"null prior uses API", types.StringNull(), "per $groupBy", "per $groupBy"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := keepDollarEscaped(c.prior, c.api).ValueString(); got != c.want {
				t.Fatalf("got %q; want %q", got, c.want)
			}
		})
	}
}
