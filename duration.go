package main

import (
	"math"
	"regexp"
	"strconv"
	"time"
)

var durationPartRegex = regexp.MustCompile(`([0-9]+)(ms|s|m|h|d|w|y)`)

var durationUnits = map[string]time.Duration{
	"ms": time.Millisecond,
	"s":  time.Second,
	"m":  time.Minute,
	"h":  time.Hour,
	"d":  24 * time.Hour,
	"w":  7 * 24 * time.Hour,
	"y":  365 * 24 * time.Hour,
}

// parseDuration parses an AxonOps duration ("30s", "24h", "1d", "1h30m").
func parseDuration(s string) (time.Duration, bool) {
	if !durationRegex.MatchString(s) {
		return 0, false
	}
	var total time.Duration
	for _, m := range durationPartRegex.FindAllStringSubmatch(s, -1) {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return 0, false
		}
		unit := durationUnits[m[2]]
		if n > int64(math.MaxInt64/unit) {
			return 0, false
		}
		part := time.Duration(n) * unit
		if total > math.MaxInt64-part {
			return 0, false
		}
		total += part
	}
	return total, true
}

// durationsEqual reports whether a and b are the same duration however
// spelled (24h, 1d, 1440m). Unparseable values fall back to string equality.
func durationsEqual(a, b string) bool {
	if a == b {
		return true
	}
	da, okA := parseDuration(a)
	db, okB := parseDuration(b)
	return okA && okB && da == db
}

// preferPriorDuration returns prior when it is equivalent to the value the
// API returned, so a reformatted value does not show as a diff.
func preferPriorDuration(prior, api string) string {
	if prior != "" && durationsEqual(prior, api) {
		return prior
	}
	return api
}
