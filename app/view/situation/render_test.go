package situation

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

func render(t *testing.T, view View) string {
	t.Helper()
	var out bytes.Buffer
	if err := LayerRail(view).Render(context.Background(), &out); err != nil {
		t.Fatalf("render: %v", err)
	}
	return out.String()
}

func tieredView() View {
	return View{Layers: []domain.SituationLayer{
		{
			ID: "sigmet.intl", Label: "SIGMET (international)", MinPlan: domain.PlanFree,
			Entitled: true, Available: true, Attribution: "NOAA", Licence: "public-domain", Count: 141,
		},
		{
			ID: "quake.usgs.m25", Label: "Earthquakes (M2.5+)", MinPlan: domain.PlanPro,
			Entitled: false, Available: true, Attribution: "USGS", Licence: "public-domain",
		},
		{
			ID: "notam.faa", Label: "NOTAMs", MinPlan: domain.PlanBusiness,
			Entitled: true, Available: false, Attribution: "FAA", Licence: "public-domain",
		},
	}}
}

func TestLayerRailOffersAnUpgradeForGatedLayers(t *testing.T) {
	html := render(t, tieredView())

	// A gated layer is shown, named, and priced — hiding it would leave the
	// customer unaware of a capability they could buy.
	if !strings.Contains(html, "Earthquakes (M2.5+)") {
		t.Error("the gated layer is not listed")
	}
	if !strings.Contains(html, "/pricing") {
		t.Error("no upgrade link")
	}
	if !strings.Contains(html, "pro") {
		t.Error("the required plan is not named")
	}
}

func TestLayerRailSeparatesLockedFromUnconfigured(t *testing.T) {
	html := render(t, tieredView())

	// An upgrade fixes one and not the other. Offering to sell a plan that
	// would not switch the layer on is worse than saying nothing.
	upgradeAt := strings.Index(html, "Upgrade to unlock")
	unconfiguredAt := strings.Index(html, "Not configured")
	if upgradeAt < 0 {
		t.Fatal("no upgrade section")
	}
	if unconfiguredAt < 0 {
		t.Fatal("no unconfigured section")
	}

	notamAt := strings.Index(html, "NOTAMs")
	if notamAt < unconfiguredAt {
		t.Error("the unconfigured layer is not under the unconfigured heading")
	}
}

func TestLayerRailShowsATogglePerEntitledLayer(t *testing.T) {
	html := render(t, tieredView())

	// Only the entitled layer gets a checkbox: a toggle for something the
	// server would refuse is a control that does nothing.
	if got := strings.Count(html, `type="checkbox"`); got != 1 {
		t.Errorf("got %d toggles, want 1", got)
	}
	if !strings.Contains(html, `value="sigmet.intl"`) {
		t.Error("the entitled layer has no toggle")
	}
	if strings.Contains(html, `value="quake.usgs.m25"`) {
		t.Error("a gated layer was given a working toggle")
	}
}

func TestLayerRailShowsObservationCounts(t *testing.T) {
	html := render(t, tieredView())

	// The count is what tells a reader a layer is live rather than merely
	// switched on.
	if !strings.Contains(html, "141") {
		t.Error("the observation count is not rendered")
	}
}

func TestAttributionBarLinksToTheLicences(t *testing.T) {
	var out bytes.Buffer
	if err := AttributionBar(tieredView()).Render(context.Background(), &out); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := out.String()

	// Naming the sources satisfies attribution; linking the terms is what lets
	// someone check what they may do with the data.
	if !strings.Contains(html, "/legal/data-sources") {
		t.Error("the attribution bar does not link to the licences page")
	}
	if !strings.Contains(html, "NOAA") {
		t.Error("the attribution bar does not credit its sources")
	}
}
