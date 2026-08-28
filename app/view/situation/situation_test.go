package situation

import (
	"encoding/json"
	"testing"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

func view() View {
	return View{Layers: []domain.SituationLayer{
		{ID: "sigmet.intl", Entitled: true, Available: true, Attribution: "NOAA Aviation Weather Center"},
		{ID: "gairmet", Entitled: true, Available: true, Attribution: "NOAA Aviation Weather Center"},
		{ID: "fire.firms", Entitled: false, Available: true, Attribution: "NASA FIRMS"},
		{ID: "notam.faa", Entitled: true, Available: false, Attribution: "FAA"},
	}}
}

func TestEntitledExcludesLockedAndUnconfiguredLayers(t *testing.T) {
	entitled := view().Entitled()
	if len(entitled) != 2 {
		t.Fatalf("got %d entitled layers, want 2", len(entitled))
	}
}

func TestLockedAndUnconfiguredAreDistinct(t *testing.T) {
	v := view()

	// An upgrade fixes one and not the other, so the rail must not conflate
	// them into a single "unavailable" state.
	locked := v.Locked()
	if len(locked) != 1 || locked[0].ID != "fire.firms" {
		t.Errorf("locked = %+v, want fire.firms", locked)
	}
	unconfigured := v.Unconfigured()
	if len(unconfigured) != 1 || unconfigured[0].ID != "notam.faa" {
		t.Errorf("unconfigured = %+v, want notam.faa", unconfigured)
	}
}

func TestAttributionsAreDeduplicated(t *testing.T) {
	credits := view().Attributions()

	// Two NOAA layers must credit NOAA once, not twice.
	if len(credits) != 3 {
		t.Fatalf("got %d credits %v, want 3", len(credits), credits)
	}
	if credits[0] != "NOAA Aviation Weather Center" {
		t.Errorf("credits[0] = %q", credits[0])
	}
}

func TestBootstrapJSONIsValidAndCarriesEveryLayer(t *testing.T) {
	var payload Bootstrap
	if err := json.Unmarshal([]byte(view().BootstrapJSON()), &payload); err != nil {
		t.Fatalf("bootstrap is not valid JSON: %v", err)
	}
	// The map reads this on first paint, and it filters by entitlement itself,
	// so the payload carries the whole catalogue rather than a subset.
	if len(payload.Layers) != 4 {
		t.Errorf("got %d layers, want 4", len(payload.Layers))
	}
}

func TestBootstrapJSONOfAnEmptyViewIsStillValid(t *testing.T) {
	var payload Bootstrap
	if err := json.Unmarshal([]byte(View{}.BootstrapJSON()), &payload); err != nil {
		t.Fatalf("empty bootstrap is not valid JSON: %v", err)
	}
	if len(payload.Layers) != 0 {
		t.Errorf("got %d layers, want 0", len(payload.Layers))
	}
}

func TestNewsViewAttributionsAreDeduplicated(t *testing.T) {
	view := NewsView{Items: []domain.SituationEvent{
		{ID: "a", Attribution: "The GDELT Project"},
		{ID: "b", Attribution: "The GDELT Project"},
		{ID: "c", Attribution: "ReliefWeb"},
		{ID: "d"},
	}}

	credits := view.Attributions()
	// The rail is swapped in on its own by htmx, so it carries its own credits
	// rather than relying on the page footer being present.
	if len(credits) != 2 {
		t.Fatalf("got %d credits %v, want 2", len(credits), credits)
	}
	if credits[0] != "The GDELT Project" {
		t.Errorf("credits[0] = %q", credits[0])
	}
}

func TestNewsViewHasMoreFollowsTheCursor(t *testing.T) {
	if (NewsView{}).HasMore() {
		t.Error("HasMore is true with no cursor")
	}
	if !(NewsView{NextCursor: "abc"}).HasMore() {
		t.Error("HasMore is false with a cursor")
	}
}
