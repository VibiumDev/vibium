package api

import (
	"errors"
	"strings"
	"testing"
)

// A not-found error on an index-addressed element must say the handle came
// from findAll and may be stale; everything else passes through (#338).
func TestStaleIndexHint(t *testing.T) {
	notFound := errors.New("timeout after 5s: element not found")

	t.Run("indexed not-found gets the findAll explanation", func(t *testing.T) {
		got := staleIndexHint(notFound, ElementParams{HasIndex: true, Index: 2})
		if !strings.Contains(got.Error(), "element 2 of a findAll result") {
			t.Fatalf("expected the findAll hint, got: %v", got)
		}
		if !errors.Is(got, notFound) {
			t.Fatal("the original error must stay wrapped")
		}
	})

	t.Run("non-indexed not-found is unchanged", func(t *testing.T) {
		if got := staleIndexHint(notFound, ElementParams{}); got != notFound {
			t.Fatalf("expected the error untouched, got: %v", got)
		}
	})

	t.Run("indexed errors that are not not-found are unchanged", func(t *testing.T) {
		other := errors.New("context is blocked by an open alert dialog")
		if got := staleIndexHint(other, ElementParams{HasIndex: true, Index: 1}); got != other {
			t.Fatalf("expected the error untouched, got: %v", got)
		}
	})

	t.Run("nil stays nil", func(t *testing.T) {
		if got := staleIndexHint(nil, ElementParams{HasIndex: true}); got != nil {
			t.Fatalf("expected nil, got: %v", got)
		}
	})
}
