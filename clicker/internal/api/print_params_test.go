package api

import (
	"reflect"
	"testing"
)

// vibium:page.pdf options must translate to browsingContext.print parameters,
// and options the caller did not set must not appear at all, so the browser's
// own defaults stay in charge (#72).
func TestPrintParams(t *testing.T) {
	t.Run("no options sends only the context", func(t *testing.T) {
		got := printParams("ctx1", nil)
		want := map[string]interface{}{"context": "ctx1"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("unrelated wire keys are ignored", func(t *testing.T) {
		got := printParams("ctx1", map[string]interface{}{"filename": "x.pdf", "context": "ctx1"})
		want := map[string]interface{}{"context": "ctx1"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("all options translate to the BiDi shapes", func(t *testing.T) {
		got := printParams("ctx1", map[string]interface{}{
			"landscape":    true,
			"scale":        0.8,
			"background":   true,
			"marginTop":    2.0,
			"marginBottom": 2.0,
			"marginLeft":   1.5,
			"marginRight":  1.5,
			"pageWidth":    29.7,
			"pageHeight":   21.0,
			"pageRanges":   []interface{}{float64(1), "3-5"},
			"shrinkToFit":  false,
		})
		want := map[string]interface{}{
			"context":     "ctx1",
			"orientation": "landscape",
			"scale":       0.8,
			"background":  true,
			"margin":      map[string]interface{}{"top": 2.0, "bottom": 2.0, "left": 1.5, "right": 1.5},
			"page":        map[string]interface{}{"width": 29.7, "height": 21.0},
			"pageRanges":  []interface{}{float64(1), "3-5"},
			"shrinkToFit": false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v\nwant %v", got, want)
		}
	})

	t.Run("landscape false is portrait, the default, so nothing is sent", func(t *testing.T) {
		got := printParams("ctx1", map[string]interface{}{"landscape": false})
		if _, ok := got["orientation"]; ok {
			t.Fatalf("landscape:false must not send an orientation, got %v", got)
		}
	})

	t.Run("partial margins send only the set sides", func(t *testing.T) {
		got := printParams("ctx1", map[string]interface{}{"marginTop": 3.0})
		want := map[string]interface{}{"context": "ctx1", "margin": map[string]interface{}{"top": 3.0}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}
