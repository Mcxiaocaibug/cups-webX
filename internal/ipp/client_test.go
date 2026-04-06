package ipp

import "testing"

func TestSidesKeywordForDuplexMode(t *testing.T) {
	tests := map[string]string{
		"":                      "one-sided",
		"one-sided":             "one-sided",
		"two-sided-long-edge":   "two-sided-long-edge",
		"two-sided-short-edge":  "two-sided-short-edge",
		"something-unsupported": "one-sided",
	}

	for input, want := range tests {
		if got := sidesKeywordForDuplexMode(input); got != want {
			t.Fatalf("mode %q: got %q, want %q", input, got, want)
		}
	}
}

func TestParseJobIdentifier(t *testing.T) {
	tests := []struct {
		input   string
		wantID  int
		wantURI string
	}{
		{input: "123", wantID: 123},
		{input: "http://localhost:631/jobs/456", wantID: 456, wantURI: "http://localhost:631/jobs/456"},
		{input: "ipp://localhost/jobs/789", wantID: 789, wantURI: "ipp://localhost/jobs/789"},
		{input: "job-xyz"},
		{input: ""},
	}

	for _, tt := range tests {
		gotID, gotURI := parseJobIdentifier(tt.input)
		if gotID != tt.wantID || gotURI != tt.wantURI {
			t.Fatalf("input %q: got (%d, %q), want (%d, %q)", tt.input, gotID, gotURI, tt.wantID, tt.wantURI)
		}
	}
}

func TestRecordStatusForJobState(t *testing.T) {
	tests := map[string]string{
		"3":                  "queued",
		"pending-held":       "queued",
		"5":                  "processing",
		"processing-stopped": "processing",
		"7":                  "cancelled",
		"aborted":            "failed",
		"9":                  "printed",
		"unknown":            "",
	}

	for input, want := range tests {
		if got := recordStatusForJobState(input); got != want {
			t.Fatalf("state %q: got %q, want %q", input, got, want)
		}
	}
}

func TestStatusDetailForJobState(t *testing.T) {
	if got := statusDetailForJobState("processing", "", []string{"job-printing"}); got != "正在打印" {
		t.Fatalf("expected translated reason, got %q", got)
	}
	if got := statusDetailForJobState("failed", "Paper jam", nil); got != "Paper jam" {
		t.Fatalf("expected state message to win, got %q", got)
	}
	if got := statusDetailForJobState("queued", "", nil); got != "任务已进入打印队列" {
		t.Fatalf("expected queued fallback detail, got %q", got)
	}
}
