package handlers

import (
	"errors"
	"testing"
)

func TestParseWindowDaysDefaultsWhenAbsent(t *testing.T) {
	t.Parallel()
	got, err := parseWindowDays("")
	if err != nil {
		t.Fatalf("parseWindowDays(\"\") error = %v", err)
	}
	if got != globeDefaultWindowDays {
		t.Fatalf("parseWindowDays(\"\") = %d, want %d", got, globeDefaultWindowDays)
	}

	if got, err := parseWindowDays("  30 "); err != nil || got != 30 {
		t.Fatalf("parseWindowDays(\" 30 \") = (%d, %v)", got, err)
	}
}

// A malformed window must be rejected rather than silently defaulting: a typo
// in a shared link should be visible, not quietly change the numbers shown.
func TestParseWindowDaysRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"0", "-1", "91", "abc", "7.5", "1e3", "٣"} {
		if _, err := parseWindowDays(raw); err == nil {
			t.Fatalf("parseWindowDays(%q) should have failed", raw)
		} else if !errors.Is(err, errInvalidWindow) {
			t.Fatalf("parseWindowDays(%q) error = %v, want errInvalidWindow", raw, err)
		}
	}

	// The bounds themselves are valid.
	for _, raw := range []string{"1", "90"} {
		if _, err := parseWindowDays(raw); err != nil {
			t.Fatalf("parseWindowDays(%q) should be accepted: %v", raw, err)
		}
	}
}

func TestGlobeErrorMessage(t *testing.T) {
	t.Parallel()
	if errInvalidWindow.Error() == "" {
		t.Fatal("errInvalidWindow should explain the accepted range")
	}
}
