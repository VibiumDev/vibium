package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestConnectFromEnvReadsHeaderLines(t *testing.T) {
	t.Setenv("VIBIUM_CONNECT_URL", "wss://grid.example.com/session")
	t.Setenv("VIBIUM_CONNECT_API_KEY", "")
	t.Setenv("VIBIUM_CONNECT_HEADERS", "Authorization: Bearer tok\nX-Region: eu")

	url, headers := connectFromEnv()
	if url != "wss://grid.example.com/session" {
		t.Errorf("url = %q", url)
	}
	if got := headers.Get("Authorization"); got != "Bearer tok" {
		t.Errorf("Authorization = %q", got)
	}
	if got := headers.Get("X-Region"); got != "eu" {
		t.Errorf("X-Region = %q", got)
	}
}

func TestConnectFromEnvHeaderLinesOverrideAPIKey(t *testing.T) {
	t.Setenv("VIBIUM_CONNECT_API_KEY", "env-key")
	t.Setenv("VIBIUM_CONNECT_HEADERS", "Authorization: Bearer flag-wins")

	_, headers := connectFromEnv()
	if got := headers.Values("Authorization"); len(got) != 1 || got[0] != "Bearer flag-wins" {
		t.Errorf("Authorization = %v, want exactly [Bearer flag-wins]", got)
	}
}

func TestHeadersEnvValueRoundTrip(t *testing.T) {
	in := http.Header{}
	in.Set("Authorization", "Bearer tok")
	in.Set("X-Region", "eu")

	t.Setenv("VIBIUM_CONNECT_API_KEY", "")
	t.Setenv("VIBIUM_CONNECT_HEADERS", headersEnvValue(in))

	_, out := connectFromEnv()
	for key := range in {
		if out.Get(key) != in.Get(key) {
			t.Errorf("%s = %q, want %q", key, out.Get(key), in.Get(key))
		}
	}
}

func TestConnectSpawnEnvCarriesSecrets(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer tok")

	env := connectSpawnEnv("ws://user:key@host/session", headers, `{"a":1}`)

	want := []string{
		"VIBIUM_CONNECT_URL=ws://user:key@host/session",
		"VIBIUM_CONNECT_HEADERS=Authorization: Bearer tok",
		`VIBIUM_CONNECT_CAPS={"a":1}`,
	}
	for _, entry := range want {
		found := false
		for _, e := range env {
			if e == entry {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("env missing %q", entry)
		}
	}
}

func TestHeadersEnvValueDeterministic(t *testing.T) {
	in := http.Header{}
	in.Set("B-Second", "2")
	in.Set("A-First", "1")

	got := headersEnvValue(in)
	if !strings.HasPrefix(got, "A-First: 1\n") {
		t.Errorf("not sorted: %q", got)
	}
}
