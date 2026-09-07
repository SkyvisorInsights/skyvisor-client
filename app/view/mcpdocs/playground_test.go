package mcpdocs

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
)

func TestPlaygroundPageRendersUsageAndGovernance(t *testing.T) {
	t.Parallel()

	// Mirrors a real Free account: reads plus a small action budget. The
	// fixture previously set the action limit to 0 and asserted "Read only",
	// which stopped being a state Free can be in once Free gained a budget —
	// so it read like documentation of Free while describing something else.
	usage := apiclient.UsageSnapshot{
		MCPReads:            17,
		MCPDailyReadLimit:   100,
		MCPActions:          0,
		MCPDailyActionLimit: 5,
		AssistantCalls:      3,
		AssistantDailyLimit: 10,
		Entitlements: apiclient.Entitlements{
			Plan: "free",
		},
	}

	var output bytes.Buffer
	if err := PlaygroundPage(usage, true, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render MCP playground: %v", err)
	}

	html := output.String()
	for _, expected := range []string{"API connected", "FREE", "17 / 100", "Actions enabled", "get_operations_dashboard"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("MCP playground HTML missing %q", expected)
		}
	}
}

// TestPlaygroundPageRendersReadOnlyGovernance keeps coverage of the read-only
// badge now that the Free fixture no longer produces it. A zero action limit is
// still valid input — an admin could set one, and the renderer must say so
// rather than imply actions are available.
func TestPlaygroundPageRendersReadOnlyGovernance(t *testing.T) {
	t.Parallel()

	usage := apiclient.UsageSnapshot{
		MCPReads:            5,
		MCPDailyReadLimit:   100,
		MCPActions:          0,
		MCPDailyActionLimit: 0,
		AssistantDailyLimit: 10,
		Entitlements: apiclient.Entitlements{
			Plan: "free",
		},
	}

	var output bytes.Buffer
	if err := PlaygroundPage(usage, true, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render MCP playground: %v", err)
	}
	if !strings.Contains(output.String(), "Read only") {
		t.Fatal("a zero action limit must render the read-only badge")
	}
}

func TestPlaygroundPageRendersUnavailableUsageState(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := PlaygroundPage(apiclient.UsageSnapshot{}, false, "Usage could not be loaded.").Render(context.Background(), &output); err != nil {
		t.Fatalf("render MCP playground: %v", err)
	}

	html := output.String()
	for _, expected := range []string{"Usage unavailable", "Usage could not be loaded."} {
		if !strings.Contains(html, expected) {
			t.Fatalf("unavailable MCP playground HTML missing %q", expected)
		}
	}
	if strings.Contains(html, "MCP reads") {
		t.Fatal("unavailable MCP playground rendered quota metrics")
	}
}
