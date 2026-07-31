package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
	"golang.org/x/sync/singleflight"
)

// AirportCoordSource is the slice of the airport repository the resolver needs.
// Narrow on purpose so tests can stub it without a database.
type AirportCoordSource interface {
	GetAirportCoordinates(ctx context.Context) ([]models.AirportCoord, error)
}

const geoRefreshTTL = 12 * time.Hour

// GeoResolver caches the IATA -> coordinate table in process.
//
// This is the only place in the system that can turn the IATA codes coming back
// from skyvisor-api into positions: the API has no geographic table, and the
// reference data lives in this service's own database.
//
// ~10k rows at ~150 B is a couple of megabytes, and the data is effectively
// static, so a per-process map beats a Redis round trip on a per-request path.
type GeoResolver struct {
	source AirportCoordSource
	ttl    time.Duration
	sf     singleflight.Group

	mu       sync.RWMutex
	byIATA   map[string]models.AirportCoord
	byICAO   map[string]models.AirportCoord
	loadedAt time.Time
}

func NewGeoResolver(source AirportCoordSource) *GeoResolver {
	return &GeoResolver{
		source: source,
		ttl:    geoRefreshTTL,
		byIATA: map[string]models.AirportCoord{},
		byICAO: map[string]models.AirportCoord{},
	}
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Warm loads the table eagerly. Safe to call from a goroutine at boot; callers
// that arrive first will block on the same singleflight load either way.
func (g *GeoResolver) Warm(ctx context.Context) error {
	_, err := g.load(ctx)
	return err
}

func (g *GeoResolver) load(ctx context.Context) (int, error) {
	if g.source == nil {
		return 0, errors.New("geo resolver: no coordinate source")
	}

	value, err, _ := g.sf.Do("load", func() (any, error) {
		coords, err := g.source.GetAirportCoordinates(ctx)
		if err != nil {
			return 0, err
		}

		byIATA := make(map[string]models.AirportCoord, len(coords))
		byICAO := make(map[string]models.AirportCoord, len(coords))
		for _, c := range coords {
			c.IATA = normalizeCode(c.IATA)
			c.ICAO = normalizeCode(c.ICAO)
			if c.IATA != "" {
				byIATA[c.IATA] = c
			}
			if c.ICAO != "" {
				byICAO[c.ICAO] = c
			}
		}

		// Swap whole maps under the write lock so readers never observe a
		// half-populated table.
		g.mu.Lock()
		g.byIATA = byIATA
		g.byICAO = byICAO
		g.loadedAt = time.Now()
		g.mu.Unlock()

		return len(byIATA), nil
	})
	if err != nil {
		return 0, err
	}
	count, _ := value.(int)
	return count, nil
}

// ensureLoaded refreshes on first use and whenever the TTL has elapsed. A failed
// refresh keeps serving the previous table rather than emptying the cache.
func (g *GeoResolver) ensureLoaded() {
	g.mu.RLock()
	loadedAt := g.loadedAt
	g.mu.RUnlock()

	if !loadedAt.IsZero() && time.Since(loadedAt) < g.ttl {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := g.load(ctx); err != nil {
		slog.Warn("airport coordinate refresh failed", "error", err, "stale_since", loadedAt)
	}
}

// Lookup resolves an IATA code. A false second return means "unknown" and must
// be surfaced to the user, never substituted with a guess.
func (g *GeoResolver) Lookup(iata string) (models.AirportCoord, bool) {
	code := normalizeCode(iata)
	if code == "" {
		return models.AirportCoord{}, false
	}
	g.ensureLoaded()

	g.mu.RLock()
	defer g.mu.RUnlock()
	coord, ok := g.byIATA[code]
	return coord, ok
}

func (g *GeoResolver) LookupICAO(icao string) (models.AirportCoord, bool) {
	code := normalizeCode(icao)
	if code == "" {
		return models.AirportCoord{}, false
	}
	g.ensureLoaded()

	g.mu.RLock()
	defer g.mu.RUnlock()
	coord, ok := g.byICAO[code]
	return coord, ok
}

// Resolve looks up many codes at once. The second return lists the codes that
// could not be resolved, deduplicated and in input order, so callers can report
// "N routes hidden" instead of quietly dropping them.
func (g *GeoResolver) Resolve(codes []string) (map[string]models.AirportCoord, []string) {
	resolved := make(map[string]models.AirportCoord, len(codes))
	unresolved := make([]string, 0)
	seen := make(map[string]struct{}, len(codes))

	if len(codes) == 0 {
		return resolved, unresolved
	}
	g.ensureLoaded()

	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, raw := range codes {
		code := normalizeCode(raw)
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}

		if coord, ok := g.byIATA[code]; ok {
			resolved[code] = coord
			continue
		}
		unresolved = append(unresolved, code)
	}
	return resolved, unresolved
}

// Stats reports cache size and age, for the globe's "unresolved" footnote and
// for health checks.
func (g *GeoResolver) Stats() (int, time.Time) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.byIATA), g.loadedAt
}

// Geo exposes the resolver. Never nil.
func (h *Service) Geo() *GeoResolver {
	return h.geo
}

// CoordLookup adapts the resolver to the narrow function the view layer takes,
// keeping database access out of templ helpers.
func (h *Service) CoordLookup() func(iata string) (float64, float64, bool) {
	return func(iata string) (float64, float64, bool) {
		coord, ok := h.geo.Lookup(iata)
		if !ok {
			return 0, 0, false
		}
		return coord.Lat, coord.Lon, true
	}
}
