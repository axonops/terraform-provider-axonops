package main

import "testing"

func TestNormalizeJSON(t *testing.T) {
	tests := []struct{ in, want string }{
		{`{"b": 1, "a": [ {"y":2,"x":1} ]}`, `{"a":[{"x":1,"y":2}],"b":1}`},
		{`{"max": 1000000, "f": 0.5}`, `{"f":0.5,"max":1000000}`},
		// Terraform's jsonencode escapes <, > and & the same way.
		{`{"q": "a<b"}`, `{"q":"a` + `\` + `u003cb"}`},
		{`not json`, `not json`},
	}
	for _, tt := range tests {
		if got := normalizeJSON([]byte(tt.in)); got != tt.want {
			t.Errorf("normalizeJSON(%s) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
