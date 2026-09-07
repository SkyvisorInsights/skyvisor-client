package mcpdocs

import (
	"fmt"
	"strings"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
)

func connectionDot(available bool) string {
	if available {
		return "bg-emerald-500 shadow-[0_0_0_4px_rgba(16,185,129,0.12)]"
	}
	return "bg-amber-500"
}

func connectionLabel(available bool) string {
	if available {
		return "API connected"
	}
	return "Usage unavailable"
}

func planLabel(usage apiclient.UsageSnapshot) string {
	if usage.Entitlements.Plan == "" {
		return "Unknown"
	}
	return strings.ToUpper(string(usage.Entitlements.Plan))
}

func usageLabel(used, limit int) string {
	if limit < 0 {
		return fmt.Sprintf("%d / ∞", used)
	}
	return fmt.Sprintf("%d / %d", used, limit)
}

func remainingLabel(used, limit int) string {
	if limit < 0 {
		return "Unlimited"
	}
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("%d remaining today", remaining)
}

func actionLabel(usage apiclient.UsageSnapshot) string {
	if usage.MCPDailyActionLimit == 0 {
		return "Read only"
	}
	return "Actions enabled"
}

// oneClickSnippet is the connect instruction for a host that speaks OAuth.
// This is the primary path: the client discovers the authorization server from
// the MCP endpoint and opens a browser, so no token is ever pasted.
func oneClickSnippet() string {
	return `Add a custom connector in your agent host and point it at:

  https://mcp.skyvisor.app

Your host opens a browser, you approve the connection, and it is done.
No token to copy.

Claude Code:
  claude mcp add --transport http skyvisor https://mcp.skyvisor.app`
}

func actionClass(usage apiclient.UsageSnapshot) string {
	if usage.MCPDailyActionLimit == 0 {
		return "bg-muted text-muted-foreground"
	}
	return "bg-primary/10 text-primary"
}

func modeClass(mode string) string {
	switch mode {
	case "action":
		return "bg-amber-500/10 text-amber-700 dark:text-amber-400"
	case "assistant":
		return "bg-primary/10 text-primary"
	default:
		return "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
	}
}

func runSnippet() string {
	return `export SKYVISOR_API_URL=http://127.0.0.1:8080
export SKYVISOR_OIDC_ACCESS_TOKEN=<access-token>
cd skyvisor-mcp && go run ./cmd/skyvisor-mcp

# Or streamable HTTP (k8s / remote hosts):
# MCP_TRANSPORT=http ADDR=0.0.0.0:8087 go run ./cmd/skyvisor-mcp
# Client: Authorization: Bearer <access-token>
# Staging: https://staging-mcp.skyvisor.app`
}

func configSnippet() string {
	return `{
  "mcpServers": {
    "skyvisor": {
      "command": "go",
      "args": ["run", "./cmd/skyvisor-mcp"],
      "cwd": "/path/to/skyvisor-mcp",
      "env": {
        "SKYVISOR_API_URL": "http://127.0.0.1:8080",
        "SKYVISOR_OIDC_ACCESS_TOKEN": "<access-token>"
      }
    }
  }
}`
}

func remoteSnippet() string {
	return `Remote (staging): https://staging-mcp.skyvisor.app
Header: Authorization: Bearer <token>
Transport: MCP streamable HTTP (MCP_TRANSPORT=http)

Prefer the one-click connector above. These manual paths remain for
scripting and for hosts that cannot perform an OAuth flow.`
}
