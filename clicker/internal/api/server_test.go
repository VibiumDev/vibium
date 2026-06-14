package api

import (
	"net/http/httptest"
	"testing"
)

func TestNewServerRejectsNonLocalOrigins(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest("GET", "http://127.0.0.1:9515/", nil)
	req.Header.Set("Origin", "https://evil.example")

	if server.upgrader.CheckOrigin(req) {
		t.Fatalf("expected cross-origin websocket request to be rejected")
	}
}

func TestNewServerDefaultsToLoopbackListenAddress(t *testing.T) {
	server := NewServer(WithPort(0))

	if got, want := server.listenAddr(), "127.0.0.1:0"; got != want {
		t.Fatalf("listenAddr() = %q, want %q", got, want)
	}
}
