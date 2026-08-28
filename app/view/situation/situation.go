// Package situation renders the global situation monitor.
package situation

import (
	"encoding/json"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

// View is everything the page needs to render without a second round trip.
type View struct {
	Layers []domain.SituationLayer

	// News is the first page of the rail, rendered with the page so a reader
	// sees stories on first paint rather than after a round trip.
	News NewsView

	// Unavailable explains why the page is empty, when it is. An empty page
	// with no explanation reads as a broken deployment.
	Unavailable string
}

// Bootstrap is the payload embedded in the page for the map to read on first
// paint, so the globe draws without waiting for a fetch.
type Bootstrap struct {
	Layers []domain.SituationLayer `json:"layers"`
}

// BootstrapJSON marshals the bootstrap payload.
//
// A marshalling failure yields an empty but valid envelope rather than an
// error: the rails are server-rendered and still useful, so a broken map is
// not worth failing the whole page for.
func (v View) BootstrapJSON() string {
	payload, err := json.Marshal(Bootstrap{Layers: v.Layers})
	if err != nil {
		return `{"layers":[]}`
	}
	return string(payload)
}

// Entitled reports the layers the caller may actually switch on.
func (v View) Entitled() []domain.SituationLayer {
	entitled := make([]domain.SituationLayer, 0, len(v.Layers))
	for _, layer := range v.Layers {
		if layer.Entitled && layer.Available {
			entitled = append(entitled, layer)
		}
	}
	return entitled
}

// Locked reports layers a higher plan would unlock, so the rail can offer an
// upgrade rather than hiding a capability the account could buy.
func (v View) Locked() []domain.SituationLayer {
	locked := make([]domain.SituationLayer, 0, len(v.Layers))
	for _, layer := range v.Layers {
		if !layer.Entitled {
			locked = append(locked, layer)
		}
	}
	return locked
}

// Unconfigured reports layers this deployment has no credential for. Distinct
// from locked: an upgrade would not help, so the rail must not offer one.
func (v View) Unconfigured() []domain.SituationLayer {
	missing := make([]domain.SituationLayer, 0, len(v.Layers))
	for _, layer := range v.Layers {
		if layer.Entitled && !layer.Available {
			missing = append(missing, layer)
		}
	}
	return missing
}

// Attributions lists each distinct source credit exactly once.
//
// Several upstreams permit reuse only with credit, so this is an obligation the
// page carries rather than a nicety. Order follows the catalogue so the list is
// stable between renders.
func (v View) Attributions() []string {
	seen := make(map[string]struct{}, len(v.Layers))
	credits := make([]string, 0, len(v.Layers))
	for _, layer := range v.Layers {
		if layer.Attribution == "" {
			continue
		}
		if _, ok := seen[layer.Attribution]; ok {
			continue
		}
		seen[layer.Attribution] = struct{}{}
		credits = append(credits, layer.Attribution)
	}
	return credits
}

// NewsView is the rail: one page of stories plus the cursor for the next.
type NewsView struct {
	Items      []domain.SituationEvent
	NextCursor string

	// Unavailable explains an empty rail. An empty list with no reason reads
	// as a broken deployment rather than a quiet news hour.
	Unavailable string
}

// Attributions lists each distinct credit in the rail exactly once.
//
// GDELT is CC-BY-NC-SA, so the credit is an obligation wherever its content is
// rendered — including a partial swapped in on its own.
func (n NewsView) Attributions() []string {
	seen := make(map[string]struct{}, len(n.Items))
	credits := make([]string, 0, 2)
	for _, item := range n.Items {
		if item.Attribution == "" {
			continue
		}
		if _, ok := seen[item.Attribution]; ok {
			continue
		}
		seen[item.Attribution] = struct{}{}
		credits = append(credits, item.Attribution)
	}
	return credits
}

// HasMore reports whether a "load more" control should be shown.
func (n NewsView) HasMore() bool { return n.NextCursor != "" }
