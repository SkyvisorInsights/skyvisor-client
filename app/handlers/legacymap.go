package handlers

import "strings"

// legacyMapPrefixes are the route trees whose templates draw an OpenLayers map
// through the global `ol` object, declared in an inline script.
//
// The Reference and Explorer surfaces still render with OpenLayers, while the
// flagship map surfaces (track, globe, situation) use MapLibre. Until those are
// consolidated onto one library, the bundle loads only where it is used.
var legacyMapPrefixes = []string{
	"/airlines",
	"/airports",
	"/flights",
	"/locations",
}

// needsLegacyMap reports whether a page must load the OpenLayers bundle.
//
// The bundle is 877 KB and is a render-blocking script, so loading it on pages
// that draw no OpenLayers map delays first paint for nothing. Matching is on
// whole path segments: "/airportsomething" is not an airports route.
func needsLegacyMap(path string) bool {
	for _, prefix := range legacyMapPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
