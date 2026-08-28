package legal

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

func renderDataSources(t *testing.T, layers []domain.SituationLayer) string {
	t.Helper()
	var out bytes.Buffer
	if err := DataSourcesPage(layers).Render(context.Background(), &out); err != nil {
		t.Fatalf("render: %v", err)
	}
	return out.String()
}

func TestDataSourcesCreditsEveryLayer(t *testing.T) {
	html := renderDataSources(t, []domain.SituationLayer{
		{ID: "sigmet.intl", Label: "SIGMET (international)", Attribution: "NOAA Aviation Weather Center", Licence: "public-domain"},
		{ID: "gdacs", Label: "Disaster alerts", Attribution: "GDACS — European Commission and UN OCHA", Licence: "CC-BY-4.0"},
		{ID: "news.gdelt", Label: "World news", Attribution: "The GDELT Project", Licence: "CC-BY-NC-SA-4.0"},
	})

	// Rendered from the live catalogue, so a layer cannot ship without its
	// credit appearing. A hand-maintained list is one that eventually is not.
	for _, credit := range []string{
		"NOAA Aviation Weather Center",
		"GDACS — European Commission and UN OCHA",
		"The GDELT Project",
	} {
		if !strings.Contains(html, credit) {
			t.Errorf("missing credit %q", credit)
		}
	}
	for _, licence := range []string{"public-domain", "CC-BY-4.0", "CC-BY-NC-SA-4.0"} {
		if !strings.Contains(html, licence) {
			t.Errorf("missing licence %q", licence)
		}
	}
}

func TestDataSourcesNamesTheNonCommercialSources(t *testing.T) {
	html := renderDataSources(t, nil)

	// These two forbid commercial use, which is why the layers built on them
	// are free on every plan. Saying so is part of complying with them.
	for _, want := range []string{"GDELT", "Open-Meteo", "non-commercial"} {
		if !strings.Contains(html, want) {
			t.Errorf("the page does not mention %q", want)
		}
	}
}

func TestDataSourcesExplainsAnEmptyCatalogue(t *testing.T) {
	html := renderDataSources(t, nil)

	// An empty table with no explanation reads as a broken page rather than a
	// deployment with the feature switched off.
	if !strings.Contains(html, "not enabled") {
		t.Error("an empty catalogue is not explained")
	}
}

func TestDataSourcesCarriesAnAccuracyWarning(t *testing.T) {
	html := renderDataSources(t, nil)

	// Aeronautical decisions must be made against the originating authority's
	// own publication. Saying so on the page that lists the sources is the
	// right place for it.
	if !strings.Contains(html, "official briefing") {
		t.Error("no accuracy warning")
	}
}
