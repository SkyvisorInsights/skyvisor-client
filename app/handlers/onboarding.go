package handlers

import (
	"net/http"

	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
	"github.com/SkyvisorInsights/Aviation-tracker/app/view/onboarding"
	"github.com/gorilla/csrf"
)

func (h *Handler) WelcomePage(w http.ResponseWriter, r *http.Request) error {
	name := ""
	if currentUser, ok := r.Context().Value(models.CtxKeyAuthUser).(*models.UserSession); ok && currentUser != nil {
		name = currentUser.Username
	}
	// Pro only. A business intent reaching the welcome page rendered a checkout
	// form posting to the single Pro price, so a bookmarked or stale
	// ?plan=business link is dropped rather than honoured.
	plan := r.URL.Query().Get("plan")
	if plan != "pro" {
		plan = ""
	}
	page := onboarding.WelcomePage(name, plan, csrf.Token(r))
	return h.CreateLayout(w, r, "Welcome", page).Render(r.Context(), w)
}
