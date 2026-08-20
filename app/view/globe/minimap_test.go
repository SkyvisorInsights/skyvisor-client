package globe

import (
	"strings"
	"testing"
)

func TestMiniProjectCorners(t *testing.T) {
	t.Parallel()

	x, y := MiniProject(-180, 90)
	closeTo(t, x, 0, 1e-9, "west edge x")
	closeTo(t, y, 0, 1e-9, "north edge y")

	x, y = MiniProject(180, -90)
	closeTo(t, x, MiniWidth(), 1e-9, "east edge x")
	closeTo(t, y, MiniHeight(), 1e-9, "south edge y")

	x, y = MiniProject(0, 0)
	closeTo(t, x, MiniWidth()/2, 1e-9, "centre x")
	closeTo(t, y, MiniHeight()/2, 1e-9, "centre y")
}

// North must be up and east must be right, the same as the main globe.
func TestMiniProjectOrientation(t *testing.T) {
	t.Parallel()

	_, northY := MiniProject(0, 45)
	_, southY := MiniProject(0, -45)
	if northY >= southY {
		t.Fatalf("north y (%v) should be above south y (%v)", northY, southY)
	}

	eastX, _ := MiniProject(45, 0)
	westX, _ := MiniProject(-45, 0)
	if eastX <= westX {
		t.Fatalf("east x (%v) should be right of west x (%v)", eastX, westX)
	}
}

func TestMiniLandPathIsWellFormed(t *testing.T) {
	t.Parallel()

	path := MiniLandPath()
	if path == "" {
		t.Fatal("MiniLandPath produced nothing")
	}
	if !strings.HasPrefix(path, "M") {
		t.Fatalf("path must start with a move, got %.16s", path)
	}
	if strings.Contains(path, "NaN") || strings.Contains(path, "Inf") {
		t.Fatal("path contains non-finite coordinates")
	}
	// Many separate rings, so many subpaths.
	if strings.Count(path, "M") < 50 {
		t.Fatalf("expected many subpaths, got %d", strings.Count(path, "M"))
	}
}

// A country spanning the antimeridian must not be drawn as a band straight
// across the inset.
func TestMiniLandPathBreaksAtTheSeam(t *testing.T) {
	t.Parallel()

	path := MiniLandPath()
	// Every drawn segment must be a short hop. Parsing the path back out is the
	// only way to assert that no line spans the full width.
	var prevX float64
	var drawing bool
	maxJump := 0.0

	for _, token := range strings.Fields(strings.ReplaceAll(strings.ReplaceAll(path, "M", " M"), "L", " L")) {
		if len(token) < 2 {
			continue
		}
		cmd, rest := token[0], token[1:]
		x := parseFloatOrZero(rest)
		switch cmd {
		case 'M':
			drawing = true
			prevX = x
		case 'L':
			if drawing {
				if jump := absFloat(x - prevX); jump > maxJump {
					maxJump = jump
				}
			}
			prevX = x
		}
	}

	// Half the inset width would be a seam-spanning line; real coastline steps
	// are a couple of units at most.
	if maxJump > MiniWidth()/2 {
		t.Fatalf("path contains a %.1f unit horizontal jump, which means the antimeridian seam was not broken", maxJump)
	}
}

func parseFloatOrZero(value string) float64 {
	var out float64
	var sign float64 = 1
	i := 0
	if i < len(value) && (value[i] == '-' || value[i] == '+') {
		if value[i] == '-' {
			sign = -1
		}
		i++
	}
	seenDot := false
	frac := 0.1
	for ; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= '0' && c <= '9':
			if seenDot {
				out += float64(c-'0') * frac
				frac /= 10
			} else {
				out = out*10 + float64(c-'0')
			}
		case c == '.':
			seenDot = true
		default:
			return sign * out
		}
	}
	return sign * out
}

func TestFormatLonLatIsFixedWidth(t *testing.T) {
	t.Parallel()

	if got := FormatLonLat(-9.1359); got != "-9.135900" {
		t.Fatalf("FormatLonLat(-9.1359) = %q", got)
	}
	if got := FormatLonLat(0); got != "0.000000" {
		t.Fatalf("FormatLonLat(0) = %q", got)
	}
	// Six decimals throughout, so the readout does not jitter in width.
	for _, value := range []float64{1, -180, 179.999999, 38.7813} {
		got := FormatLonLat(value)
		if dot := strings.IndexByte(got, '.'); dot < 0 || len(got)-dot-1 != 6 {
			t.Fatalf("FormatLonLat(%v) = %q, want six decimal places", value, got)
		}
	}
}
