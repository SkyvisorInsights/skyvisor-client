#!/usr/bin/env bash
#
# Builds the self-hosted globe basemap from Natural Earth.
#
# Natural Earth is public domain, so this carries no attribution or share-alike
# obligation (unlike an OpenStreetMap-derived basemap, which is ODbL). We still
# credit it in the map's attribution control.
#
# The globe renders between z0 and roughly z6. At that scale the entire dataset
# is smaller than a tile pyramid's index, so it ships as plain GeoJSON served as
# a static asset — no tiling toolchain, no range requests, no Git LFS. If the
# detail pages ever need z8+, revisit this with tippecanoe + PMTiles and keep
# the style's source URLs pointing at the new artifact.
#
# Requires: ogr2ogr (GDAL), curl, unzip.
# Run manually via `make basemap`; the output is committed. This is NOT part of
# `make assets` or CI — the source data changes about once a year.

set -euo pipefail

OUT_DIR="app/static/geo"
VERSION="v1"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

CDN="https://naciscdn.org/naturalearth"

# Simplification tolerance in degrees. ~0.01deg is roughly 1km, well below one
# screen pixel at the zoom levels the globe uses.
SIMPLIFY="0.01"
# 3 decimal places is ~110m of precision, which is far finer than z6 can show
# and roughly halves the file size versus the default 15 digits.
PRECISION="3"

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "error: $1 is required but not installed" >&2; exit 1; }
}
need ogr2ogr
need curl
need unzip

fetch() {
  local url="$1" name="$2"
  echo "  fetching $name"
  curl -fsSL "$url" -o "$WORK_DIR/$name.zip"
  unzip -qo "$WORK_DIR/$name.zip" -d "$WORK_DIR/$name"
}

# $1 source layer, $2 output basename, $3 SQL select, $4 simplify tolerance
# ("" to skip), $5 "no-rfc7946" to skip RFC 7946 output
convert() {
  local name="$1" out="$2" sql="$3" simplify="${4-}" rfc="${5-}"
  local args=()
  if [ -n "$simplify" ]; then
    args+=(-simplify "$simplify")
  fi
  if [ "$rfc" != "no-rfc7946" ]; then
    args+=(-lco RFC7946=YES)
  fi
  echo "  converting $name -> $out.json${simplify:+ (simplify $simplify)}"
  ogr2ogr -f GeoJSON \
    "$OUT_DIR/$out-$VERSION.json" \
    "$WORK_DIR/$name/$name.shp" \
    -dialect SQLITE -sql "$sql" \
    ${args[@]+"${args[@]}"} \
    -lco COORDINATE_PRECISION="$PRECISION" \
    -lco WRITE_NAME=NO
}

mkdir -p "$OUT_DIR"

echo "Natural Earth -> $OUT_DIR"

# Country polygons. Used for both the land fill and the boundary lines, so we
# only ship one geometry set rather than a separate coastline file.
fetch "$CDN/110m/cultural/ne_110m_admin_0_countries.zip" ne_110m_admin_0_countries
convert ne_110m_admin_0_countries world-land \
  "SELECT geometry, NAME AS name, ISO_A2 AS iso FROM ne_110m_admin_0_countries" \
  "$SIMPLIFY"

# 15-degree graticule, for the faint lat/lon grid on the globe.
#
# Deliberately NOT simplified. Graticule lines are straight in lon/lat but must
# curve when projected onto a globe, so they need their intermediate vertices;
# simplifying them collapses each line to its endpoints (and GDAL degrades the
# result to MultiPoint), which renders as chords instead of arcs.
#
# RFC 7946 is also skipped here: it splits parallels that span the antimeridian
# into a GeometryCollection containing a degenerate Point, and MapLibre's
# geojson-vt does not render GeometryCollection.
fetch "$CDN/110m/physical/ne_110m_graticules_15.zip" ne_110m_graticules_15
convert ne_110m_graticules_15 world-graticule \
  "SELECT geometry FROM ne_110m_graticules_15" "" no-rfc7946

echo
echo "Done. Sizes:"
for f in "$OUT_DIR"/*-$VERSION.json; do
  printf "  %-44s %6s KB raw  %6s KB gzip\n" \
    "$(basename "$f")" \
    "$(( $(wc -c < "$f") / 1024 ))" \
    "$(( $(gzip -c "$f" | wc -c) / 1024 ))"
done
echo
echo "Artifacts are versioned in the filename so they can be cached immutably."
echo "Bump VERSION in this script when the data changes."
