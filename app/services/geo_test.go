package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SkyvisorInsights/Aviation-tracker/app/models"
)

type stubCoordSource struct {
	mu     sync.Mutex
	coords []models.AirportCoord
	err    error
	calls  atomic.Int32
}

func (s *stubCoordSource) GetAirportCoordinates(_ context.Context) ([]models.AirportCoord, error) {
	s.calls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	out := make([]models.AirportCoord, len(s.coords))
	copy(out, s.coords)
	return out, nil
}

func (s *stubCoordSource) set(coords []models.AirportCoord, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.coords = coords
	s.err = err
}

func sampleCoords() []models.AirportCoord {
	return []models.AirportCoord{
		{IATA: "LIS", ICAO: "LPPT", Lat: 38.7813, Lon: -9.1359, Name: "Lisbon"},
		{IATA: "JFK", ICAO: "KJFK", Lat: 40.6413, Lon: -73.7781, Name: "John F Kennedy"},
	}
}

func TestGeoResolverLookupHitAndMiss(t *testing.T) {
	t.Parallel()
	resolver := NewGeoResolver(&stubCoordSource{coords: sampleCoords()})

	coord, ok := resolver.Lookup("LIS")
	if !ok {
		t.Fatal("Lookup(LIS) missed")
	}
	if coord.Lat != 38.7813 || coord.Lon != -9.1359 {
		t.Fatalf("Lookup(LIS) = %+v", coord)
	}

	if _, ok := resolver.Lookup("ZZZ"); ok {
		t.Fatal("Lookup(ZZZ) should miss")
	}
	if _, ok := resolver.Lookup(""); ok {
		t.Fatal("Lookup(\"\") should miss")
	}
}

func TestGeoResolverNormalizesCodes(t *testing.T) {
	t.Parallel()
	resolver := NewGeoResolver(&stubCoordSource{coords: sampleCoords()})

	for _, input := range []string{"lis", "  LIS ", "Lis", "\tlIs\n"} {
		if _, ok := resolver.Lookup(input); !ok {
			t.Fatalf("Lookup(%q) missed; codes should be trimmed and upper-cased", input)
		}
	}
}

func TestGeoResolverLookupICAO(t *testing.T) {
	t.Parallel()
	resolver := NewGeoResolver(&stubCoordSource{coords: sampleCoords()})

	coord, ok := resolver.LookupICAO("kjfk")
	if !ok || coord.IATA != "JFK" {
		t.Fatalf("LookupICAO(kjfk) = %+v, ok=%v", coord, ok)
	}
}

// Rows with no ICAO must not create a "" key that would match a blank query.
func TestGeoResolverIgnoresBlankCodes(t *testing.T) {
	t.Parallel()
	resolver := NewGeoResolver(&stubCoordSource{coords: []models.AirportCoord{
		{IATA: "LIS", ICAO: "", Lat: 38.7813, Lon: -9.1359},
	}})

	if _, ok := resolver.LookupICAO(""); ok {
		t.Fatal("blank ICAO should never resolve")
	}
}

func TestGeoResolverResolveReportsUnresolved(t *testing.T) {
	t.Parallel()
	resolver := NewGeoResolver(&stubCoordSource{coords: sampleCoords()})

	resolved, unresolved := resolver.Resolve([]string{"LIS", "ZZZ", "jfk", "LIS", "QQQ", "  "})

	if len(resolved) != 2 {
		t.Fatalf("resolved = %d, want 2: %+v", len(resolved), resolved)
	}
	if _, ok := resolved["JFK"]; !ok {
		t.Fatal("resolved map should be keyed by normalized code")
	}
	// Deduplicated, in input order, blanks skipped.
	if len(unresolved) != 2 || unresolved[0] != "ZZZ" || unresolved[1] != "QQQ" {
		t.Fatalf("unresolved = %v, want [ZZZ QQQ]", unresolved)
	}
}

func TestGeoResolverResolveEmptyInput(t *testing.T) {
	t.Parallel()
	source := &stubCoordSource{coords: sampleCoords()}
	resolver := NewGeoResolver(source)

	resolved, unresolved := resolver.Resolve(nil)
	if len(resolved) != 0 || len(unresolved) != 0 {
		t.Fatalf("empty input gave resolved=%v unresolved=%v", resolved, unresolved)
	}
	if calls := source.calls.Load(); calls != 0 {
		t.Fatalf("empty input should not hit the source, got %d calls", calls)
	}
}

// A database outage must degrade to "unknown", never to a stale hardcoded table
// and never to a coordinate the data does not support.
func TestGeoResolverSourceFailureYieldsUnknown(t *testing.T) {
	t.Parallel()
	resolver := NewGeoResolver(&stubCoordSource{err: errors.New("db down")})

	if _, ok := resolver.Lookup("LIS"); ok {
		t.Fatal("Lookup must miss when the source is unavailable")
	}
	resolved, unresolved := resolver.Resolve([]string{"LIS", "JFK"})
	if len(resolved) != 0 {
		t.Fatalf("resolved = %+v, want empty", resolved)
	}
	if len(unresolved) != 2 {
		t.Fatalf("unresolved = %v, want both codes reported", unresolved)
	}
	if count, loadedAt := resolver.Stats(); count != 0 || !loadedAt.IsZero() {
		t.Fatalf("Stats() = (%d, %v), want (0, zero time)", count, loadedAt)
	}
}

func TestGeoResolverCachesUntilTTL(t *testing.T) {
	t.Parallel()
	source := &stubCoordSource{coords: sampleCoords()}
	resolver := NewGeoResolver(source)

	for i := 0; i < 25; i++ {
		if _, ok := resolver.Lookup("LIS"); !ok {
			t.Fatal("Lookup(LIS) missed")
		}
	}
	if calls := source.calls.Load(); calls != 1 {
		t.Fatalf("source called %d times, want 1 (results should be cached)", calls)
	}
}

func TestGeoResolverReloadsAfterTTL(t *testing.T) {
	t.Parallel()
	source := &stubCoordSource{coords: sampleCoords()}
	resolver := NewGeoResolver(source)
	resolver.ttl = time.Nanosecond // expire immediately

	if _, ok := resolver.Lookup("LIS"); !ok {
		t.Fatal("first lookup missed")
	}
	source.set([]models.AirportCoord{{IATA: "AMS", Lat: 52.3105, Lon: 4.7639}}, nil)
	time.Sleep(2 * time.Millisecond)

	if _, ok := resolver.Lookup("AMS"); !ok {
		t.Fatal("resolver did not pick up refreshed data after the TTL elapsed")
	}
	if _, ok := resolver.Lookup("LIS"); ok {
		t.Fatal("stale entry survived the refresh; the map should be swapped wholesale")
	}
}

// A failed refresh should keep serving the last good table rather than emptying it.
func TestGeoResolverKeepsStaleDataWhenRefreshFails(t *testing.T) {
	t.Parallel()
	source := &stubCoordSource{coords: sampleCoords()}
	resolver := NewGeoResolver(source)
	resolver.ttl = time.Nanosecond

	if _, ok := resolver.Lookup("LIS"); !ok {
		t.Fatal("first lookup missed")
	}
	source.set(nil, errors.New("db down"))
	time.Sleep(2 * time.Millisecond)

	if _, ok := resolver.Lookup("LIS"); !ok {
		t.Fatal("a failed refresh must not discard the previously loaded table")
	}
}

func TestGeoResolverConcurrentLookupsAndRefresh(t *testing.T) {
	t.Parallel()
	source := &stubCoordSource{coords: sampleCoords()}
	resolver := NewGeoResolver(source)
	resolver.ttl = time.Millisecond

	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 60; j++ {
				resolver.Lookup("LIS")
				resolver.LookupICAO("KJFK")
				resolver.Resolve([]string{"LIS", "ZZZ"})
				resolver.Stats()
			}
		}()
	}
	wg.Wait()
}

func TestGeoResolverWarm(t *testing.T) {
	t.Parallel()
	source := &stubCoordSource{coords: sampleCoords()}
	resolver := NewGeoResolver(source)

	if err := resolver.Warm(context.Background()); err != nil {
		t.Fatalf("Warm() error = %v", err)
	}
	count, loadedAt := resolver.Stats()
	if count != 2 {
		t.Fatalf("Stats() count = %d, want 2", count)
	}
	if loadedAt.IsZero() {
		t.Fatal("Stats() loadedAt is zero after Warm()")
	}

	failing := NewGeoResolver(&stubCoordSource{err: errors.New("db down")})
	if err := failing.Warm(context.Background()); err == nil {
		t.Fatal("Warm() should surface source errors")
	}
}
