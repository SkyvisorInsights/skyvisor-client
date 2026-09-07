package oauthconsent

import (
	"strings"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

// clientLabel prefers the registered name and falls back to the identifier, so
// the heading always names something concrete.
func clientLabel(client apiclient.OAuthClientInfo) string {
	if name := strings.TrimSpace(client.ClientName); name != "" {
		return name
	}
	if id := strings.TrimSpace(client.ClientID); id != "" {
		return id
	}
	return "this agent"
}

// grantsActions reports whether the requested scope includes acting, which is
// what decides whether the page warns about writes.
func grantsActions(scope string) bool {
	for _, field := range strings.Fields(scope) {
		if field == domain.OAuthScopeAct {
			return true
		}
	}
	return false
}
