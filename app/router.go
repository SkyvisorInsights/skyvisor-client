package app

import (
	"context"
	"crypto/sha256"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/apiclient"
	"github.com/SkyvisorInsights/Aviation-tracker/app/auth"
	"github.com/SkyvisorInsights/Aviation-tracker/app/handlers"
	"github.com/SkyvisorInsights/Aviation-tracker/app/repository"
	"github.com/SkyvisorInsights/Aviation-tracker/app/services"
	thinkingorbs "github.com/FACorreiaa/Thinking-orbs-go/components"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/riandyrn/otelchi"
)

//go:embed static
var staticFS embed.FS

func setupBusinessComponents(pool *pgxpool.Pool, redisClient *redis.Client, validate *validator.Validate,
	sessionSecret []byte, cookieSecure bool, oidcClient *auth.Client,
	apiClient *apiclient.Client) (*handlers.Handler, *repository.MiddlewareRepository) {
	sessionStore := sessions.NewCookieStore(sessionSecret)
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	// Business components
	airlineRepo := repository.NewAirlineRepository(pool)
	airportRepo := repository.NewAirportRepository(pool)
	locationRepo := repository.NewLocationsRepository(pool)
	flightsRepo := repository.NewFlightsRepository(pool)
	authRepo := repository.NewAccountRepository(pool, redisClient, validate, sessionStore)

	// Middleware
	middleware := &repository.MiddlewareRepository{
		Pgpool:      pool,
		RedisClient: redisClient,
		Validator:   validate,
		Sessions:    sessionStore,
	}

	// Service
	service := services.NewService(airlineRepo, airportRepo, locationRepo, flightsRepo, authRepo, oidcClient, apiClient)

	// Handler
	handler := handlers.NewHandler(service, sessionStore, pool, redisClient, oidcClient)

	return handler, middleware
}

func Router(pool *pgxpool.Pool, sessionSecret []byte, cookieSecure bool, redisClient *redis.Client,
	oidcClient *auth.Client, apiClient *apiclient.Client) http.Handler {
	validate := validator.New()
	translator, _ := ut.New(en.New(), en.New()).GetTranslator("en")
	if err := enTranslations.RegisterDefaultTranslations(validate, translator); err != nil {
		slog.Error("Error registering translations", "error", err)
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(otelchi.Middleware("skyvisor-web", otelchi.WithChiRoutes(r)))
	if sentry.CurrentHub().Client() != nil {
		// Repanic so Recoverer (below) still converts the panic into a 500.
		r.Use(sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle)
	}
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RedirectSlashes)
	r.Use(securityHeaders)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			slog.WarnContext(ctx, "readiness check failed", "dependency", "postgres", "error", err)
			http.Error(w, "postgres unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.WarnContext(ctx, "readiness check failed", "dependency", "redis", "error", err)
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// Static files
	r.Handle("/static/*", http.FileServer(http.FS(staticFS)))
	if scriptFS, err := fs.Sub(thinkingorbs.ScriptFS, "orb"); err != nil {
		slog.Error("thinking orbs script fs", "error", err)
	} else {
		r.Handle("/thinking-orbs/js/*", http.StripPrefix("/thinking-orbs/js/", http.FileServer(http.FS(scriptFS))))
	}
	r.Get("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		file, err := staticFS.ReadFile("static/favicon.ico")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Content-Type", http.DetectContentType(file))
		if _, err := w.Write(file); err != nil {
			slog.Error("write favicon", "error", err)
		}
	})

	h, authMiddleware := setupBusinessComponents(pool, redisClient, validate, sessionSecret, cookieSecure, oidcClient, apiClient)

	// Public routes, authentication is optional
	r.With(authMiddleware.AuthMiddleware).Get("/", handler(h.Homepage))
	r.With(authMiddleware.AuthMiddleware).Get("/track", handler(h.TrackFlight))
	r.With(authMiddleware.AuthMiddleware).Get("/terms", handler(h.TermsPage))
	r.With(authMiddleware.AuthMiddleware).Get("/privacy", handler(h.PrivacyPage))
	r.With(authMiddleware.AuthMiddleware).Get("/pricing", handler(h.PricingPage))
	// Public pickup share pages do not require a session.
	r.With(authMiddleware.AuthMiddleware).Get("/share/{token}", handler(h.SharePage))

	// OIDC callback must stay reachable regardless of auth state.
	r.With(authMiddleware.AuthMiddleware).Get("/auth/callback", handler(h.AuthCallback))

	// Routes that shouldn't be available to authenticated users
	r.Group(func(guest chi.Router) {
		guest.Use(authMiddleware.AuthMiddleware)
		guest.Use(authMiddleware.RedirectIfAuth)
		guest.Get("/login", handler(h.LoginStart))
		guest.Get("/register", handler(h.RegisterStart))
	})

	// Authenticated routes
	r.Group(func(auth chi.Router) {
		auth.Use(authMiddleware.AuthMiddleware)
		auth.Use(authMiddleware.RequireAuth)

		auth.Post("/logout", handler(h.Logout))
		auth.Get("/welcome", handler(h.WelcomePage))
		auth.Get("/dashboard", handler(h.DashboardPage))
		auth.Get("/settings", handler(h.SettingsPage))
		auth.Post("/settings/alerts", handler(h.SettingsAlerts))
		auth.Post("/settings/tokens", handler(h.SettingsTokensCreate))
		auth.Post("/settings/tokens/{id}/delete", handler(h.SettingsTokensRevoke))

		auth.Route("/trips", func(trips chi.Router) {
			trips.Get("/", handler(h.TripsPage))
			trips.Get("/timeline", handler(h.TripsTimeline))
			trips.Post("/", handler(h.TripsCreate))
			trips.Post("/import", handler(h.TripsImport))
			trips.Post("/import/pdf", handler(h.TripsImportPDF))
			trips.Post("/assistant", handler(h.TripsAssistant))
			trips.Post("/what-if", handler(h.TripsWhatIf))
			trips.Post("/{id}/delete", handler(h.TripsDelete))
		})

		// Browser SSE proxy: forwards the account's live flight events.
		auth.Get("/events", handler(h.EventsStream))

		auth.Route("/watches", func(watches chi.Router) {
			watches.Get("/", handler(h.WatchesPage))
			watches.Post("/", handler(h.WatchesCreate))
			watches.Post("/{id}/share", handler(h.WatchesShare))
			watches.Post("/{id}/delete", handler(h.WatchesDelete))
		})
		auth.Post("/billing/checkout", handler(h.BillingCheckout))
		auth.Post("/billing/portal", handler(h.BillingPortal))
		auth.Get("/analytics", handler(h.AnalyticsPage))
		auth.Get("/analytics/export", handler(h.AnalyticsExport))
		auth.Get("/logistics", handler(h.LogisticsPage))
		auth.Post("/logistics/team", handler(h.LogisticsCreateTeam))
		auth.Post("/logistics/team/join", handler(h.LogisticsJoinTeam))
		auth.Get("/mcp", handler(h.MCPPlaygroundPage))
		auth.Route("/operations/cases", func(operations chi.Router) {
			operations.Get("/", handler(h.OperationalCasesPage))
			operations.Post("/", handler(h.OperationalCasesCreate))
			operations.Get("/{id}", handler(h.OperationalCasePage))
			operations.Post("/{id}/decisions", handler(h.OperationalDecisionCreate))
			operations.Post("/{id}/decisions/{decisionID}/action", handler(h.OperationalDecisionAction))
			operations.Post("/{id}/decisions/{decisionID}/outcome", handler(h.OperationalDecisionOutcome))
		})

		auth.Route("/airlines", func(airlines chi.Router) {
			airlines.Get("/airline", handler(h.AirlineMainPage))
			airlines.Get("/airline/location", handler(h.AirlineLocationPage))
			airlines.Get("/airline/{airline_name}", handler(h.AirlineDetailsPage))
			airlines.Get("/aircraft", handler(h.AirlineAircraftPage))
			airlines.Get("/airplane", handler(h.AirlineAirplanePage))
			airlines.Get("/tax", handler(h.AirlineTaxPage))
		})

		auth.Route("/locations", func(locations chi.Router) {
			locations.Get("/city", handler(h.CityMainPage))
			locations.Get("/city/map", handler(h.CityLocationsPage))
			locations.Get("/city/details/{city_id}", handler(h.CityDetailsPage))
			locations.Get("/country", handler(h.CountryMainPage))
			locations.Get("/country/map", handler(h.CountryLocationPage))
			locations.Get("/country/details/{country_name}", handler(h.CountryDetailsPage))
		})

		auth.Route("/airports", func(airports chi.Router) {
			airports.Get("/", handler(h.AirportPage))
			airports.Get("/board", handler(h.AirportBoardPage))
			airports.Get("/{iata}/board", handler(h.AirportBoardPage))
			airports.Get("/locations", handler(h.AirportLocationPage))
			airports.Get("/details/{airport_id}", handler(h.AirportDetailsPage))
		})

		auth.Route("/flights", func(flights chi.Router) {
			flights.Get("/tracker", handler(h.LiveTrackerPage))
			flights.Get("/flight/live", handler(h.LiveFlightsPage))
			flights.Get("/flight/location/air/live", handler(h.LiveFlightsLocationsPage))
			flights.Get("/flight/location", handler(h.FlightsLocation))
			flights.Get("/flight/status/location/{flight_status}", handler(h.FlightsLocationsByStatus))
			flights.Get("/flight/status/{flight_status}", handler(h.FilteredFlightsPage))
			flights.Get("/flight", handler(h.AllFlightsPage))
			flights.Get("/flight/{flight_number}", handler(h.DetailedFlightsPage))
		})
	})

	csrfKey := sha256.Sum256(append(append([]byte(nil), sessionSecret...), []byte(":csrf")...))
	csrfMiddleware := csrf.Protect(
		csrfKey[:],
		csrf.Secure(cookieSecure),
		csrf.HttpOnly(true),
		csrf.Path("/"),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.FieldName("csrf_token"),
		// gorilla/csrf >= 1.7.3 enforces Origin/Referer checks that Safari
		// trips over (null or absent Origin on same-site posts). Trust our own
		// hosts explicitly.
		csrf.TrustedOrigins([]string{
			"127.0.0.1:6968", "127.0.0.1:6969", "localhost:6968", "localhost:6969",
			"skyvisor.app", "www.skyvisor.app", "staging.skyvisor.app",
		}),
		// Respond with HTML, never text/plain: Safari downloads text/plain
		// responses to top-level form POSTs instead of rendering them.
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.WarnContext(r.Context(), "csrf rejection",
				"path", r.URL.Path, "reason", csrf.FailureReason(r),
				"origin", r.Header.Get("Origin"), "referer", r.Header.Get("Referer"))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Form expired - SkyVisor Insight</title></head><body style="font-family:system-ui,sans-serif;display:grid;place-items:center;min-height:100dvh;margin:0;background:#0b1220;color:#e2e8f0"><main style="text-align:center;padding:2rem"><h1 style="font-size:1.25rem;margin:0 0 .5rem">That form expired</h1><p style="color:#94a3b8;margin:0 0 1.25rem">Go back and try again; the page will issue a fresh token.</p><a href="/" style="color:#60a5fa">Return to SkyVisor</a></main></body></html>`))
		})),
	)

	handlerChain := csrfMiddleware(r)
	if !cookieSecure {
		// Local development runs plain HTTP; skip the Referer origin
		// allow-list check that otherwise 403s Safari.
		plain := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			handlerChain.ServeHTTP(w, csrf.PlaintextHTTPRequest(req))
		})
		return plain
	}
	return handlerChain
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "geolocation=(self), camera=(), microphone=()")
		next.ServeHTTP(w, r)
	})
}

func handler(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			slog.Error("Error handling request", "error", err)
		}
	}
}
