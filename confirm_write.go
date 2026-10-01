package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// The AxonOps API acknowledges writes before they are visible to reads: a
// POST/PUT can return 200 while an immediate GET of the same collection does
// not yet include the change. Every Create/Update therefore confirms its write
// by polling the read endpoint until the change is observed, instead of
// trusting the write's status code or a single read-back.
//
// Declared as vars so tests can shorten them.
var (
	confirmWriteTimeout     = 30 * time.Second
	confirmWriteBaseBackoff = 250 * time.Millisecond
	confirmWriteMaxBackoff  = 2 * time.Second
)

// errWriteNotConfirmed is returned (wrapped) when a write is never observed
// on the read endpoint within confirmWriteTimeout.
var errWriteNotConfirmed = errors.New("write was acknowledged but not observed on read")

// confirmWrite polls check until it reports found, with exponential backoff
// capped at confirmWriteMaxBackoff, for about confirmWriteTimeout (the last
// read can land slightly after it, as the bound is checked before sleeping
// and a read itself takes time). Read
// errors are treated as transient and retried; the last one is included in
// the timeout error. what describes the object for log and error messages,
// e.g. `Slack integration "ops"`.
func confirmWrite[T any](ctx context.Context, what string, check func(context.Context) (T, bool, error)) (T, error) {
	var zero T
	deadline := time.Now().Add(confirmWriteTimeout)
	backoff := confirmWriteBaseBackoff
	var lastErr error

	for attempt := 1; ; attempt++ {
		v, found, err := check(ctx)
		if err == nil && found {
			if attempt > 1 {
				tflog.Debug(ctx, "Write confirmed after retry", map[string]interface{}{
					"object":   what,
					"attempts": attempt,
				})
			}
			return v, nil
		}
		lastErr = err

		if time.Now().Add(backoff).After(deadline) {
			break
		}
		tflog.Debug(ctx, "Write not yet visible, retrying", map[string]interface{}{
			"object":  what,
			"attempt": attempt,
			"backoff": backoff.String(),
		})
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("confirming %s: %w", what, ctx.Err())
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, confirmWriteMaxBackoff)
	}

	if lastErr != nil {
		return zero, fmt.Errorf("%s: %w after %s (last read error: %v)", what, errWriteNotConfirmed, confirmWriteTimeout, lastErr)
	}
	return zero, fmt.Errorf("%s: %w after %s", what, errWriteNotConfirmed, confirmWriteTimeout)
}

// confirmIntegration waits until the integration described by payload is
// listed by the API with every non-sensitive param matching what was sent.
// Sensitive params are skipped because the API masks them on read.
func confirmIntegration(ctx context.Context, c *axonopsClient.AxonopsHttpClient, clusterType, clusterName string, payload axonopsClient.IntegrationPayload, sensitiveParams ...string) (*axonopsClient.IntegrationDefinition, error) {
	name := payload.Params["name"]
	what := fmt.Sprintf("%s integration %q", payload.Type, name)
	return confirmWrite(ctx, what, func(ctx context.Context) (*axonopsClient.IntegrationDefinition, bool, error) {
		integrations, err := c.GetIntegrations(ctx, clusterType, clusterName)
		if err != nil {
			return nil, false, err
		}
		def := axonopsClient.FindIntegrationByNameAndType(integrations, name, payload.Type)
		if def == nil {
			return nil, false, nil
		}
		for k, want := range payload.Params {
			if slices.Contains(sensitiveParams, k) {
				continue
			}
			if def.Params[k] != want {
				return nil, false, nil
			}
		}
		return def, true, nil
	})
}

// confirmAlertRule waits until an alert rule named rule.Alert, matching kind,
// is listed with the expression, duration, operator, thresholds and
// description that were sent. Filters, the summary and the widget URL are not
// compared: they are derived or may be rewritten server-side.
func confirmAlertRule(ctx context.Context, c *axonopsClient.AxonopsHttpClient, clusterType, clusterName string, rule axonopsClient.MetricAlertRule, kind func(axonopsClient.MetricAlertRule) bool) (*axonopsClient.MetricAlertRule, error) {
	what := fmt.Sprintf("alert rule %q", rule.Alert)
	return confirmWrite(ctx, what, func(ctx context.Context) (*axonopsClient.MetricAlertRule, bool, error) {
		rules, err := c.GetAlertRules(ctx, clusterType, clusterName)
		if err != nil {
			return nil, false, err
		}
		found := findAlertRuleByName(rules, rule.Alert, kind)
		if found == nil ||
			found.Operator != rule.Operator ||
			found.WarningValue != rule.WarningValue ||
			found.CriticalValue != rule.CriticalValue ||
			found.Expr != rule.Expr ||
			found.For != rule.For ||
			found.Annotations.Description != rule.Annotations.Description {
			return nil, false, nil
		}
		return found, true, nil
	})
}

// confirmHealthcheck waits until a healthcheck of kind ("http", "tcp" or
// "shell") named name is listed with the interval and timeout that were sent.
func confirmHealthcheck(ctx context.Context, c *axonopsClient.AxonopsHttpClient, clusterType, clusterName, kind, name, interval, timeout string) error {
	what := fmt.Sprintf("%s healthcheck %q", kind, name)
	_, err := confirmWrite(ctx, what, func(ctx context.Context) (struct{}, bool, error) {
		checks, err := c.GetHealthchecks(ctx, clusterType, clusterName)
		if err != nil || checks == nil {
			return struct{}{}, false, err
		}
		match := func(n, i, t string) bool { return n == name && i == interval && t == timeout }
		switch kind {
		case "http":
			for _, h := range checks.HTTPChecks {
				if match(h.Name, h.Interval, h.Timeout) {
					return struct{}{}, true, nil
				}
			}
		case "tcp":
			for _, h := range checks.TCPChecks {
				if match(h.Name, h.Interval, h.Timeout) {
					return struct{}{}, true, nil
				}
			}
		case "shell":
			for _, h := range checks.ShellChecks {
				if match(h.Name, h.Interval, h.Timeout) {
					return struct{}{}, true, nil
				}
			}
		}
		return struct{}{}, false, nil
	})
	return err
}
