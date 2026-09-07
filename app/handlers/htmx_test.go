package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The header format changed with the htmx 4 upgrade, and the failure is silent:
// an unrecognised target makes a handler answer with a full page where a
// fragment was asked for, which reads as a layout bug rather than a header one.
func TestHxTargetIDAcceptsEveryHeaderSpellingHtmxHasUsed(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		header string
		want   string
	}{
		{"htmx 4 sends tag and id", "div#flight-result", "flight-result"},
		{"htmx 2 sent a bare id", "flight-result", "flight-result"},
		{"a hand-written selector", "#flight-result", "flight-result"},
		{"surrounding whitespace", "  div#globe-panels  ", "globe-panels"},
		{"absent header", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				r.Header.Set("HX-Target", tt.header)
			}
			if got := hxTargetID(r); got != tt.want {
				t.Fatalf("hxTargetID(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

// globePartial picks the fragment to render, so it has to recognise the target
// htmx actually reports rather than only the id it used to report.
func TestGlobePartialRecognisesTheHtmx4TargetHeader(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		header string
		want   string
	}{
		{"div#globe-panels", "globe-panels"},
		{"globe-panels", "globe-panels"},
		{"section#globe-view", "globe"},
		{"div#something-else", ""},
	} {
		r := httptest.NewRequest(http.MethodGet, "/globe", nil)
		r.Header.Set("HX-Target", tt.header)
		if got := globePartial(r); got != tt.want {
			t.Fatalf("globePartial(HX-Target: %q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}
