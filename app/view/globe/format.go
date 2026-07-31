package globe

import (
	"fmt"
	"strconv"
	"strings"
)

// emDash is what an unknown value renders as. Distinguishing "we have no data"
// from "the value is zero" is the whole point of these helpers.
const emDash = "—"

// Callsign formats a flight number as the label shown on the globe:
// TP1363 -> TP-1363. Numbers that do not split cleanly are returned as-is
// rather than mangled.
func Callsign(flightNumber string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(flightNumber))
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "-") {
		return trimmed
	}

	// The carrier prefix is the leading run of non-digits, which handles both
	// two-letter codes (TP1363) and alphanumeric ones (4U9525 -> 4U-9525).
	split := -1
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			// A leading digit is part of the carrier code (4U), so only split
			// once at least one character has been consumed.
			if i == 0 {
				continue
			}
			split = i
			break
		}
	}
	if split <= 0 || split >= len(trimmed) {
		return trimmed
	}

	prefix, suffix := trimmed[:split], trimmed[split:]
	// A carrier code is two or three characters; anything else is not a flight
	// number we recognise, so leave it alone.
	if len(prefix) < 2 || len(prefix) > 3 {
		return trimmed
	}
	return prefix + "-" + suffix
}

// Count renders an integer, or an em dash when the value is unknown.
func Count(value int, known bool) string {
	if !known {
		return emDash
	}
	return strconv.Itoa(value)
}

// Percent renders a percentage to one decimal place.
func Percent(value float64, known bool) string {
	if !known {
		return emDash
	}
	return strconv.FormatFloat(value, 'f', 1, 64) + "%"
}

// Minutes renders a delay in minutes.
func Minutes(value float64, known bool) string {
	if !known {
		return emDash
	}
	return strconv.FormatFloat(value, 'f', 0, 64) + "m"
}

// Delta describes a change against a previous window.
type Delta struct {
	Label string
	// Good reports whether the movement is in the desired direction, which
	// differs per metric: a falling delay is good, a falling on-time rate is not.
	Good  bool
	Known bool
}

// DeltaFrom compares two windows. inverted marks metrics where lower is better.
//
// A delta is only rendered when both windows have data — showing a change
// against a window we never measured would be an invented number.
func DeltaFrom(current, previous float64, hasCurrent, hasPrevious, inverted bool) Delta {
	if !hasCurrent || !hasPrevious {
		return Delta{Label: emDash}
	}

	change := current - previous
	if change == 0 {
		return Delta{Label: "0%", Good: true, Known: true}
	}

	sign := "+"
	if change < 0 {
		sign = "−" // U+2212, so the minus aligns with digits
	}

	magnitude := change
	if magnitude < 0 {
		magnitude = -magnitude
	}

	good := change > 0
	if inverted {
		good = change < 0
	}

	return Delta{
		Label: fmt.Sprintf("%s%s", sign, strconv.FormatFloat(magnitude, 'f', 1, 64)),
		Good:  good,
		Known: true,
	}
}

// RiskLabel is the human name for a risk bucket.
func RiskLabel(risk string) string {
	switch risk {
	case RiskRisk:
		return "At risk"
	case RiskWatch:
		return "Watch"
	case RiskOK:
		return "On track"
	default:
		return emDash
	}
}

// RouteLabel renders a route as "LIS → JFK".
func RouteLabel(dep, arr string) string {
	dep, arr = normalizeIATA(dep), normalizeIATA(arr)
	if dep == "" || arr == "" {
		return emDash
	}
	return dep + " → " + arr
}

// UnresolvedNote explains how much of the picture is missing. Empty when
// nothing is missing, so the UI shows no footnote at all.
func UnresolvedNote(unresolved Unresolved) string {
	if unresolved.Count == 0 {
		return ""
	}
	codes := strings.Join(unresolved.Codes, ", ")
	if unresolved.Count == 1 {
		return fmt.Sprintf("1 airport hidden — no coordinates for %s.", codes)
	}
	return fmt.Sprintf("%d airports hidden — no coordinates for %s.", unresolved.Count, codes)
}
