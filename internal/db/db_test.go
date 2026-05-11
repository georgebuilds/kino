package db

import (
	"testing"
	"time"
)

func TestParseTime_Variants(t *testing.T) {
	// Full RFC3339 with milliseconds.
	got := parseTime("2025-01-15T12:34:56.789Z")
	want := time.Date(2025, 1, 15, 12, 34, 56, 789_000_000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("RFC3339+millis: got %v, want %v", got, want)
	}

	// Date-only.
	got = parseTime("2025-01-15")
	want = time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("date-only: got %v, want %v", got, want)
	}

	// Empty string should return zero time, no panic.
	got = parseTime("")
	if !got.IsZero() {
		t.Fatalf("empty: got %v, want zero time", got)
	}

	// Malformed string should return zero time, no panic.
	got = parseTime("not-a-date")
	if !got.IsZero() {
		t.Fatalf("malformed: got %v, want zero time", got)
	}
}
