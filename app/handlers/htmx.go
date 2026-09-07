package handlers

import (
	"net/http"
	"strings"
)

// hxTargetID reports the id of the element htmx is swapping into, without the
// tag name or a leading '#'.
//
// The header's format changed with htmx 4: htmx 2 sent a bare id, htmx 4 sends
// "tag#id" (for example "div#flight-result"). Comparisons here are against an
// id, so the shape is normalised in one place rather than at each call site — a
// handler that matched only the htmx 2 spelling would stop recognising its own
// partial and silently answer with the wrong fragment.
func hxTargetID(r *http.Request) string {
	target := strings.TrimSpace(r.Header.Get("HX-Target"))
	if i := strings.LastIndex(target, "#"); i >= 0 {
		return target[i+1:]
	}
	return target
}
