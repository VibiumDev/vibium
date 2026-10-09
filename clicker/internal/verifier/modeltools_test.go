package verifier

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeDispatcher struct {
	dispatched map[string]interface{}
	probed     []string
	secret     bool
}

func (f *fakeDispatcher) Dispatch(ctx context.Context, name string, args map[string]interface{}) (Observation, error) {
	f.dispatched = args
	return Observation{Text: "ok"}, nil
}

func (f *fakeDispatcher) ProbeSecret(ctx context.Context, selector, name string) (bool, error) {
	f.probed = append(f.probed, selector)
	return f.secret, nil
}

// Both surfaces must present the same tools with the same schemas; the only
// permitted difference is the no-@refs note on selector tools for surfaces
// that do not resolve refs (#571).
func TestModelToolsAgreeAcrossSurfaces(t *testing.T) {
	withRefs, noRefs := ModelTools(true), ModelTools(false)
	if len(withRefs) != len(noRefs) {
		t.Fatalf("tool counts differ: %d vs %d", len(withRefs), len(noRefs))
	}
	for i := range withRefs {
		if withRefs[i].Name != noRefs[i].Name {
			t.Fatalf("tool order differs at %d: %s vs %s", i, withRefs[i].Name, noRefs[i].Name)
		}
		a, _ := json.Marshal(withRefs[i].Parameters)
		b, _ := json.Marshal(noRefs[i].Parameters)
		if string(a) != string(b) {
			t.Errorf("%s: parameters differ between surfaces", withRefs[i].Name)
		}
		props := withRefs[i].Parameters["properties"].(map[string]interface{})
		_, selector := props["selector"]
		if selector && !strings.Contains(noRefs[i].Description, "no @refs") {
			t.Errorf("%s: selector tool without refs must say so", noRefs[i].Name)
		}
		if strings.Contains(withRefs[i].Description, "no @refs") {
			t.Errorf("%s: refs surface must not carry the no-refs note", withRefs[i].Name)
		}
	}
}

// Every model tool is pinned to the existing page with no file output or
// annotation; the trims must hold for whatever toolschema grows next.
func TestModelToolsPinToThePage(t *testing.T) {
	for _, tool := range ModelTools(true) {
		props := tool.Parameters["properties"].(map[string]interface{})
		for _, banned := range []string{"page", "filename", "annotate", "fullPage"} {
			if _, ok := props[banned]; ok {
				t.Errorf("%s exposes %q to the model", tool.Name, banned)
			}
		}
	}
}

// The action timeout is capped to the time left before the run deadline on
// every surface; the SDK surface used to pin it to 5s regardless (#571).
func TestExecuteCapsTimeoutToDeadline(t *testing.T) {
	d := &fakeDispatcher{}
	e := NewModelToolExecutor(d, ToolPolicy{}, true)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := e.Execute(ctx, "browser_click", map[string]interface{}{"selector": "#a", "timeout": float64(30000)}); err != nil {
		t.Fatal(err)
	}
	got, ok := d.dispatched["timeout"].(float64)
	if !ok || got > 2000 || got <= 0 {
		t.Fatalf("timeout = %v, want capped to the <=2s left before the deadline", d.dispatched["timeout"])
	}
}

// A press without a selector targets the focused element, so the password
// probe must still run — with the empty selector the probe script reads as
// document.activeElement. The SDK surface used to skip this probe (#571).
func TestExecuteProbesPressWithoutSelector(t *testing.T) {
	d := &fakeDispatcher{secret: true}
	e := NewModelToolExecutor(d, ToolPolicy{}, true)
	obs, err := e.Execute(context.Background(), "browser_press", map[string]interface{}{"key": "a"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.probed, []string{""}) {
		t.Fatalf("probed = %v, want one probe with the empty selector", d.probed)
	}
	if d.dispatched != nil {
		t.Fatal("a secret target must not be dispatched")
	}
	if !strings.Contains(obs.Text, "Password fields are unavailable") {
		t.Fatalf("observation must explain the refusal, got %q", obs.Text)
	}
}

// With CredentialInput policy (Run), fill and type into a password field are
// allowed; reading tools still refuse.
func TestCredentialPolicyAllowsWriteNotRead(t *testing.T) {
	d := &fakeDispatcher{secret: true}
	e := NewModelToolExecutor(d, ToolPolicy{CredentialInput: true}, true)
	if _, err := e.Execute(context.Background(), "browser_fill", map[string]interface{}{"selector": "#pw", "text": "s3cret"}); err != nil {
		t.Fatal(err)
	}
	if d.dispatched == nil {
		t.Fatal("credential input policy must allow filling a password field")
	}
	d.dispatched = nil
	obs, err := e.Execute(context.Background(), "browser_get_value", map[string]interface{}{"selector": "#pw"})
	if err != nil {
		t.Fatal(err)
	}
	if d.dispatched != nil || !strings.Contains(obs.Text, "not_completed") {
		t.Fatalf("reading a password field must refuse with Run guidance, got %q", obs.Text)
	}
}
