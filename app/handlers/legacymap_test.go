package handlers

import "testing"

func TestNeedsLegacyMapOnExplorerRoutes(t *testing.T) {
	// Every surface whose templates call ol.Map must still receive the bundle.
	paths := []string{
		"/airlines/airline",
		"/airlines/airline/TAP",
		"/airports",
		"/airports/board",
		"/flights/flight",
		"/flights/tracker",
		"/locations/city",
		"/locations/country/PT",
	}
	for _, path := range paths {
		if !needsLegacyMap(path) {
			t.Errorf("needsLegacyMap(%q) = false, want true: the page draws an OpenLayers map", path)
		}
	}
}

func TestSkipsLegacyMapOnFlagshipRoutes(t *testing.T) {
	// These surfaces render with MapLibre or no map at all, so the 877 KB
	// OpenLayers bundle is dead weight ahead of first paint.
	paths := []string{
		"/",
		"/track",
		"/globe",
		"/situation",
		"/dashboard",
		"/watches",
		"/trips",
		"/pricing",
		"/settings",
		"/analytics",
		"/operations/cases",
	}
	for _, path := range paths {
		if needsLegacyMap(path) {
			t.Errorf("needsLegacyMap(%q) = true, want false: no OpenLayers map on this page", path)
		}
	}
}

func TestNeedsLegacyMapMatchesWholeSegments(t *testing.T) {
	// A prefix match must not catch an unrelated route that merely starts with
	// the same letters.
	if needsLegacyMap("/airportsomething") {
		t.Error(`needsLegacyMap("/airportsomething") = true, want false`)
	}
	if needsLegacyMap("/flightsxyz") {
		t.Error(`needsLegacyMap("/flightsxyz") = true, want false`)
	}
}
