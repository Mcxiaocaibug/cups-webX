package main

import "testing"

func TestNormalizeDuplexMode(t *testing.T) {
	tests := map[string]string{
		"":                      "one-sided",
		"false":                 "one-sided",
		"one-sided":             "one-sided",
		"true":                  "two-sided-long-edge",
		"two-sided-long-edge":   "two-sided-long-edge",
		"two-sided-short-edge":  "two-sided-short-edge",
		"something-unsupported": "one-sided",
	}

	for input, want := range tests {
		if got := normalizeDuplexMode(input); got != want {
			t.Fatalf("input %q: got %q, want %q", input, got, want)
		}
	}
}
