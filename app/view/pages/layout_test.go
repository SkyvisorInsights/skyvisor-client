package pages

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
)

func renderLayout(t *testing.T, layout models.LayoutTempl) string {
	t.Helper()
	if layout.Content == nil {
		layout.Content = templ.NopComponent
	}
	var out strings.Builder
	if err := LayoutPage(layout).Render(context.Background(), &out); err != nil {
		t.Fatalf("render layout: %v", err)
	}
	return out.String()
}

func TestLayoutLoadsOpenLayersOnlyWhenRequested(t *testing.T) {
	withMap := renderLayout(t, models.LayoutTempl{Title: "Airports", LegacyMap: true})
	if !strings.Contains(withMap, "/static/js/ol.js") {
		t.Fatal("explorer page did not load the OpenLayers bundle; its inline ol.Map script will fail")
	}

	withoutMap := renderLayout(t, models.LayoutTempl{Title: "Track"})
	if strings.Contains(withoutMap, "/static/js/ol.js") {
		t.Fatal("page with no OpenLayers map still loads the 877 KB bundle")
	}
}

func TestLayoutAlwaysLoadsSharedScripts(t *testing.T) {
	// theme.js must stay render-blocking to avoid a theme flash, and app.js
	// carries htmx and Alpine for every page.
	for _, legacyMap := range []bool{true, false} {
		body := renderLayout(t, models.LayoutTempl{Title: "Any", LegacyMap: legacyMap})
		for _, script := range []string{"/static/js/theme.js", "/static/js/app.js"} {
			if !strings.Contains(body, script) {
				t.Errorf("LegacyMap=%v: missing %s", legacyMap, script)
			}
		}
	}
}
