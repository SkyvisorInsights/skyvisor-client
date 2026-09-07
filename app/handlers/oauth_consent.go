package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	oauthconsent "github.com/SkyvisorInsights/Aviation-tracker/app/view/oauthconsent"
	"github.com/SkyvisorInsights/skyvisor-go-shared/domain"
)

// consentLoginRedirect sends an unauthenticated visitor to sign in and back to
// the same authorization request. The query string is the request, so it has
// to survive the round trip.
func consentLoginRedirect(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, "/login?return_to="+url.QueryEscape(target), http.StatusSeeOther)
}

// OAuthConsentPage asks the signed-in user whether to grant a registered MCP
// client access to their account.
//
// The API validated the client and redirect_uri before redirecting the browser
// here, and validates them again on approval. This page therefore treats its
// own query parameters as untrusted display input: the client's name is
// fetched from the API rather than read from the URL.
func (h *Handler) OAuthConsentPage(w http.ResponseWriter, r *http.Request) error {
	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" || h.service.API() == nil {
		consentLoginRedirect(w, r)
		return nil
	}
	q := r.URL.Query()
	clientID := strings.TrimSpace(q.Get("client_id"))
	if clientID == "" {
		return h.renderConsentError(w, r, "This connection request is missing its client identifier. Start again from your agent.")
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	client, err := h.service.API().DescribeOAuthClient(ctx, accessToken, clientID)
	if err != nil {
		return h.renderConsentError(w, r, friendlyAPIError(err, "That agent is not registered with SkyVisor. Start again from your agent."))
	}

	scope := strings.TrimSpace(q.Get("scope"))
	if scope == "" {
		scope = domain.OAuthScopeRead
	}
	return h.CreateLayout(w, r, "Connect an agent", oauthconsent.ConsentPage(oauthconsent.ConsentRequest{
		Client:        client,
		RedirectURI:   strings.TrimSpace(q.Get("redirect_uri")),
		CodeChallenge: strings.TrimSpace(q.Get("code_challenge")),
		State:         q.Get("state"),
		Scope:         scope,
		Resource:      strings.TrimSpace(q.Get("resource")),
	})).Render(r.Context(), w)
}

// OAuthConsentSubmit records the decision. Approval asks the API to mint an
// authorization code and returns the browser to the agent; denial returns the
// standard access_denied error to the same place, so the agent learns the
// outcome rather than hanging.
func (h *Handler) OAuthConsentSubmit(w http.ResponseWriter, r *http.Request) error {
	accessToken, err := h.apiAccessToken(r)
	if err != nil || accessToken == "" || h.service.API() == nil {
		http.Redirect(w, r, "/login?return_to=/mcp", http.StatusSeeOther)
		return nil
	}
	if err := r.ParseForm(); err != nil {
		return h.renderConsentError(w, r, "That connection request could not be read. Start again from your agent.")
	}

	input := apiclient.ApproveOAuthGrant{
		ClientID:      strings.TrimSpace(r.PostFormValue("client_id")),
		RedirectURI:   strings.TrimSpace(r.PostFormValue("redirect_uri")),
		CodeChallenge: strings.TrimSpace(r.PostFormValue("code_challenge")),
		State:         r.PostFormValue("state"),
		Scope:         strings.TrimSpace(r.PostFormValue("scope")),
		Resource:      strings.TrimSpace(r.PostFormValue("resource")),
	}

	if r.PostFormValue("decision") != "approve" {
		h.denyConsent(w, r, accessToken, input)
		return nil
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	approval, err := h.service.API().ApproveOAuthGrant(ctx, accessToken, input)
	if err != nil {
		return h.renderConsentError(w, r, friendlyAPIError(err, "That connection could not be approved. Start again from your agent."))
	}
	http.Redirect(w, r, approval.RedirectTo, http.StatusSeeOther)
	return nil
}

// denyConsent returns the standard access_denied error to the agent so it
// learns the outcome instead of hanging.
//
// The redirect target must be checked against the client's registered URIs
// first. Every value on this form arrived in the query string of whatever link
// opened the page, so an attacker can point redirect_uri anywhere; bouncing to
// it unchecked would turn the consent page into an open redirector on a
// trusted origin, triggered by the user clicking Cancel — the cautious choice.
// Being absolute and parseable is not the same as being registered.
//
// The approve path needs no equivalent check: the API re-validates redirect_uri
// against the registered set before returning anywhere to send the browser.
func (h *Handler) denyConsent(w http.ResponseWriter, r *http.Request, accessToken string, input apiclient.ApproveOAuthGrant) {
	if !h.deniedRedirectAllowed(r, accessToken, input) {
		http.Redirect(w, r, "/mcp", http.StatusSeeOther)
		return
	}
	target, err := url.Parse(input.RedirectURI)
	if err != nil {
		http.Redirect(w, r, "/mcp", http.StatusSeeOther)
		return
	}
	q := target.Query()
	q.Set("error", "access_denied")
	q.Set("error_description", "The user declined the connection request")
	if input.State != "" {
		q.Set("state", input.State)
	}
	target.RawQuery = q.Encode()
	http.Redirect(w, r, target.String(), http.StatusSeeOther)
}

// deniedRedirectAllowed reports whether the target is registered to the client,
// matching exactly as the API does. Anything unresolvable is refused: failing
// closed here costs the agent a tidy error and costs an attacker the redirect.
func (h *Handler) deniedRedirectAllowed(r *http.Request, accessToken string, input apiclient.ApproveOAuthGrant) bool {
	if input.ClientID == "" || input.RedirectURI == "" || h.service.API() == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	client, err := h.service.API().DescribeOAuthClient(ctx, accessToken, input.ClientID)
	if err != nil {
		return false
	}
	for _, registered := range client.RedirectURIs {
		if registered == input.RedirectURI {
			return true
		}
	}
	return false
}

// renderConsentError shows a dead end on our own page rather than redirecting,
// because a request that failed validation has no redirect target we trust.
func (h *Handler) renderConsentError(w http.ResponseWriter, r *http.Request, message string) error {
	w.WriteHeader(http.StatusBadRequest)
	return h.CreateLayout(w, r, "Connect an agent", oauthconsent.ConsentPage(oauthconsent.ConsentRequest{
		Error: message,
		Scope: domain.OAuthScopeRead,
	})).Render(r.Context(), w)
}
