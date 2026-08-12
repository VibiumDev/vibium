package api

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The in-view center math needs a live rect and viewport, so it lives in JS.
// These tests run the production strings through node against a stubbed DOM
// rather than reimplementing the arithmetic in Go, where a copy could agree with
// itself while the browser did something else.

// runNode writes src to a temp file and returns its stdout.
func runNode(t *testing.T, src string) string {
	t.Helper()
	bin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping JS evaluation test")
	}
	path := filepath.Join(t.TempDir(), "eval.js")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write temp script: %v", err)
	}
	out, err := exec.Command(bin, path).CombinedOutput()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}
	return string(out)
}

// TestInViewCenterJS pins the clipping: the center of the element rect
// intersected with the viewport, floored to a coordinate BiDi input can reach.
func TestInViewCenterJS(t *testing.T) {
	cases := []struct {
		name                string
		x, y, width, height float64
		vw, vh              int
		wantX, wantY        int
		wantNoVisibleArea   bool
	}{
		// The reported scrim: 884x5314 in a 901x981 viewport. chromedriver clicks
		// it at y=490; the full-rect center was y=2657, off-screen (#340).
		{name: "taller than the viewport", x: 0, y: 0, width: 884, height: 5314,
			vw: 901, vh: 981, wantX: 442, wantY: 490},
		{name: "wider than the viewport", x: 0, y: 0, width: 2400, height: 200,
			vw: 800, vh: 600, wantX: 400, wantY: 100},
		// An element that fits keeps its plain center: no behavior change.
		{name: "fits in the viewport", x: 100, y: 200, width: 80, height: 40,
			vw: 800, vh: 600, wantX: 140, wantY: 220},
		{name: "straddles the top edge", x: 0, y: -100, width: 200, height: 400,
			vw: 800, vh: 600, wantX: 100, wantY: 150},
		// Flooring keeps the probe reachable at the far edge: the last row of a
		// 600px viewport is 599, not 600.
		{name: "one pixel visible at the bottom edge", x: 0, y: 599, width: 200, height: 200,
			vw: 800, vh: 600, wantX: 100, wantY: 599},
		{name: "fractional geometry", x: 10.5, y: 20.25, width: 100.5, height: 50.5,
			vw: 800, vh: 600, wantX: 60, wantY: 45},
		{name: "entirely off-screen", x: 0, y: -500, width: 200, height: 200,
			vw: 800, vh: 600, wantNoVisibleArea: true},
		{name: "zero area", x: 100, y: 100, width: 0, height: 50,
			vw: 800, vh: 600, wantNoVisibleArea: true},
	}

	specs := make([]map[string]float64, 0, len(cases))
	for _, c := range cases {
		specs = append(specs, map[string]float64{
			"x": c.x, "y": c.y, "w": c.width, "h": c.height,
			"vw": float64(c.vw), "vh": float64(c.vh),
		})
	}
	specsJSON, err := json.Marshal(specs)
	if err != nil {
		t.Fatalf("marshal cases: %v", err)
	}

	var got []*PointInfo
	stdout := strings.TrimSpace(runNode(t, `globalThis.window = { innerWidth: 0, innerHeight: 0 };
`+inViewCenterJS+`
console.log(JSON.stringify(`+string(specsJSON)+`.map(s => {
	window.innerWidth = s.vw; window.innerHeight = s.vh;
	return inViewCenter({ x: s.x, y: s.y, width: s.w, height: s.h });
})));
`))
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("parse node output %q: %v", stdout, err)
	}
	if len(got) != len(cases) {
		t.Fatalf("got %d results for %d cases", len(got), len(cases))
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := got[i]
			if c.wantNoVisibleArea {
				if p != nil {
					t.Fatalf("got point (%d, %d), want null so the check reports no visible area", p.X, p.Y)
				}
				return
			}
			if p == nil {
				t.Fatalf("want (%d, %d), got null", c.wantX, c.wantY)
			}
			if p.X != c.wantX || p.Y != c.wantY {
				t.Errorf("in-view center = (%d, %d), want (%d, %d)", p.X, p.Y, c.wantX, c.wantY)
			}
			if p.X < 0 || p.Y < 0 || p.X >= c.vw || p.Y >= c.vh {
				t.Errorf("(%d, %d) is outside the %dx%d viewport, where elementFromPoint returns null",
					p.X, p.Y, c.vw, c.vh)
			}
		})
	}
}

// checkBodyHarness wraps the production check body in a callable function and
// stubs the DOM it touches, including the CSSOM rule the bug hinged on:
// elementFromPoint returns null outside the viewport, however solidly the
// element covers the screen. Only chkEvents runs, so no styles are needed.
const checkBodyHarness = `
const probes = [];
globalThis.window = { innerWidth: 0, innerHeight: 0 };
globalThis.document = {
	elementFromPoint(x, y) {
		probes.push([x, y]);
		if (x < 0 || y < 0 || x >= window.innerWidth || y >= window.innerHeight) return null;
		return globalThis.__hit;
	},
};

function makeEl(tag, id, shadowRoot) {
	return { tagName: tag.toUpperCase(), id: id, shadowRoot: shadowRoot || null, contains: () => false };
}

// el and rect are parameters because the body reads them from the enclosing
// scope, exactly as it does inside the page script.
function runCheck(el, rect, chkEvents) {
	const chkVisible = false, chkEnabled = false, chkEditable = false;
`

// TestReceivesEventsCheckBody runs the real check body over the stub, covering
// the branch each outcome takes — including shadow-root descent, which the clamp
// must leave working.
func TestReceivesEventsCheckBody(t *testing.T) {
	cases := []struct {
		name string
		rect [4]float64
		hit  string // what elementFromPoint resolves to inside the viewport
		want string // "" means the check passes
	}{
		{name: "element covering the viewport passes", rect: [4]float64{0, 0, 800, 2000}, hit: "self"},
		{name: "descendant on top passes", rect: [4]float64{0, 0, 800, 2000}, hit: "descendant"},
		{name: "shadow root resolving to the element passes", rect: [4]float64{10, 20, 100, 50}, hit: "shadow-self"},
		{name: "shadow root resolving elsewhere is obscured", rect: [4]float64{10, 20, 100, 50}, hit: "shadow-other",
			want: "element is obscured by <span#inner> at in-view center (60, 45)"},
		{name: "overlay is obscured", rect: [4]float64{0, 0, 800, 2000}, hit: "other",
			want: "element is obscured by <div#lid> at in-view center (400, 300)"},
		{name: "nothing painted at the probe point", rect: [4]float64{100, 200, 80, 40}, hit: "none",
			want: "no element at in-view center (140, 220)"},
		{name: "off-screen element has no visible area", rect: [4]float64{0, -500, 200, 200}, hit: "self",
			want: "element has no visible area in the viewport"},
	}

	specs := make([]map[string]interface{}, 0, len(cases))
	for _, c := range cases {
		specs = append(specs, map[string]interface{}{
			"hit":  c.hit,
			"rect": map[string]float64{"x": c.rect[0], "y": c.rect[1], "width": c.rect[2], "height": c.rect[3]},
		})
	}
	specsJSON, err := json.Marshal(specs)
	if err != nil {
		t.Fatalf("marshal cases: %v", err)
	}

	var got []struct {
		Raw    *string  `json:"raw"`
		Probes [][2]int `json:"probes"`
	}
	stdout := strings.TrimSpace(runNode(t, checkBodyHarness+actionabilityCheckBody()+`
	return null; // every check passed
}

window.innerWidth = 800; window.innerHeight = 600;
const target = makeEl('div', 'target');
const kid = makeEl('span', 'kid');
target.contains = (other) => other === kid;
console.log(JSON.stringify(`+string(specsJSON)+`.map(s => {
	probes.length = 0;
	const inner = makeEl('span', 'inner');
	const host = makeEl('my-card', 'host', { elementFromPoint: () => (s.hit === 'shadow-self' ? target : inner) });
	globalThis.__hit = {
		self: target, descendant: kid, other: makeEl('div', 'lid'),
		'shadow-self': host, 'shadow-other': host, none: null,
	}[s.hit];
	const raw = runCheck(target, s.rect, true);
	return { raw: raw === undefined ? null : raw, probes: probes.slice() };
})));
`))
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("parse node output %q: %v", stdout, err)
	}
	if len(got) != len(cases) {
		t.Fatalf("got %d results for %d cases", len(got), len(cases))
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.want == "" {
				if got[i].Raw != nil {
					t.Fatalf("check failed with %s, want pass", *got[i].Raw)
				}
				if len(got[i].Probes) == 0 {
					t.Fatal("check passed without hit-testing anything")
				}
				for _, p := range got[i].Probes {
					if p[0] < 0 || p[1] < 0 || p[0] >= 800 || p[1] >= 600 {
						t.Errorf("probed (%d, %d), outside the 800x600 viewport", p[0], p[1])
					}
				}
				return
			}
			if got[i].Raw == nil {
				t.Fatalf("check passed, want failure %q", c.want)
			}
			var res actionableResult
			if err := json.Unmarshal([]byte(*got[i].Raw), &res); err != nil {
				t.Fatalf("parse check result %q: %v", *got[i].Raw, err)
			}
			if res.Status != "failed" || res.Check != "receivesEvents" {
				t.Errorf("got status %q check %q, want failed/receivesEvents", res.Status, res.Check)
			}
			if res.Reason != c.want {
				t.Errorf("reason = %q, want %q", res.Reason, c.want)
			}
		})
	}
}

// TestActionabilityScriptsReportThePoint checks both generated scripts parse and
// hand the probed point back to Go — and only when receivesEvents ran, so fill,
// select and scroll keep their previous coordinates.
func TestActionabilityScriptsReportThePoint(t *testing.T) {
	bin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping JS syntax check")
	}
	dir := t.TempDir()

	for name, ep := range map[string]ElementParams{
		"css":      {Selector: "#scrim"},
		"semantic": {Role: "button", Text: "Go"},
	} {
		script, _ := buildActionableScript(ep, ClickChecks)
		path := filepath.Join(dir, name+".js")
		if err := os.WriteFile(path, []byte("const fn = "+script+";\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if out, err := exec.Command(bin, "--check", path).CombinedOutput(); err != nil {
			t.Fatalf("%s actionability script is not valid JS: %v\n%s", name, err, out)
		}
		if !strings.Contains(script, "const inView = chkEvents ? inViewCenter(rect) : null") {
			t.Errorf("%s script computes a point even when receivesEvents is not checked", name)
		}
		if !strings.Contains(script, "point: inView") {
			t.Errorf("%s script does not report the probed point to the caller", name)
		}
	}
}

func TestElementInfoInputPoint(t *testing.T) {
	// A scrim taller than the viewport: the box center is off-screen, so input
	// must use the point the check probed.
	tall := ElementInfo{
		Box:   BoxInfo{X: 0, Y: 0, Width: 800, Height: 2000},
		Point: &PointInfo{X: 400, Y: 300},
	}
	if x, y := tall.InputPoint(); x != 400 || y != 300 {
		t.Errorf("InputPoint() = (%d, %d), want the probed (400, 300)", x, y)
	}

	// No point (checks that skip receivesEvents, or Force): unchanged behavior.
	plain := ElementInfo{Box: BoxInfo{X: 100, Y: 200, Width: 80, Height: 40}}
	if x, y := plain.InputPoint(); x != 140 || y != 220 {
		t.Errorf("InputPoint() = (%d, %d), want the box center (140, 220)", x, y)
	}

	// (0, 0) is a real point, not a missing one.
	origin := ElementInfo{
		Box:   BoxInfo{X: -100, Y: -100, Width: 200, Height: 200},
		Point: &PointInfo{X: 0, Y: 0},
	}
	if x, y := origin.InputPoint(); x != 0 || y != 0 {
		t.Errorf("InputPoint() = (%d, %d), want (0, 0)", x, y)
	}
}
