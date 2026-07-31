// Package geo exposes the committed Natural Earth basemap geometry to Go code.
//
// The same files are served to the browser as static assets for the WebGL map;
// this package parses them once so the server can render the no-JS fallback
// globe. Regenerate them with `make basemap`.
package geo

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed world-land-v1.json
var landJSON []byte

// Ring is a closed polygon boundary as [lon, lat] pairs.
type Ring [][2]float64

var (
	landOnce sync.Once
	landData []Ring
)

type featureCollection struct {
	Features []struct {
		Geometry struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

// Land returns the country outlines as flat rings, suitable for projecting.
//
// Interior rings (lakes) are included as separate rings; at globe zoom the
// distinction is not visible and treating them uniformly keeps the projector
// simple.
func Land() []Ring {
	landOnce.Do(func() {
		var fc featureCollection
		if err := json.Unmarshal(landJSON, &fc); err != nil {
			return
		}
		for _, feature := range fc.Features {
			switch feature.Geometry.Type {
			case "Polygon":
				var polygon [][][2]float64
				if err := json.Unmarshal(feature.Geometry.Coordinates, &polygon); err != nil {
					continue
				}
				for _, ring := range polygon {
					landData = append(landData, ring)
				}
			case "MultiPolygon":
				var multi [][][][2]float64
				if err := json.Unmarshal(feature.Geometry.Coordinates, &multi); err != nil {
					continue
				}
				for _, polygon := range multi {
					for _, ring := range polygon {
						landData = append(landData, ring)
					}
				}
			}
		}
	})
	return landData
}
