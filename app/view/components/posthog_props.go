package components

import "encoding/json"

// PostHogProperties marshals event properties to a JSON object string for
// PostHogCapture. A marshalling failure yields an empty object rather than an
// error: analytics must never be the reason a page fails to render.
func PostHogProperties(props map[string]string) string {
	if len(props) == 0 {
		return "{}"
	}
	encoded, err := json.Marshal(props)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
