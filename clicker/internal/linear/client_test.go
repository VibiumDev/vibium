package linear

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateIssueAndComment(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		var req gqlRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Fatal(err)
		}
		switch {
		case strings.Contains(req.Query, "teams"):
			_, _ = w.Write([]byte(`{"data":{"teams":{"nodes":[{"id":"team-1","key":"ENG"}]}}}`))
		case strings.Contains(req.Query, "issueCreate"):
			input, _ := req.Variables["input"].(map[string]any)
			if input["teamId"] != "team-1" {
				t.Fatalf("team %v", input["teamId"])
			}
			desc, _ := input["description"].(string)
			if !strings.Contains(desc, "Signed by Vibium") {
				t.Fatalf("unsigned: %s", desc)
			}
			_, _ = w.Write([]byte(`{"data":{"issueCreate":{"success":true,"issue":{"id":"i1","identifier":"ENG-9","url":"https://linear.app/acme/issue/ENG-9","title":"t"}}}}`))
		case strings.Contains(req.Query, "commentCreate"):
			_, _ = w.Write([]byte(`{"data":{"commentCreate":{"success":true,"comment":{"id":"c1","url":"https://linear.app/c1"}}}}`))
		default:
			t.Fatalf("query %s", req.Query)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := Config{APIKey: "lin_api_test", TeamKey: "ENG", Endpoint: srv.URL, HTTP: srv.Client()}
	issue, err := cfg.CreateIssue(context.Background(), "title", "body\n\n## Signed by Vibium\n")
	if err != nil {
		t.Fatal(err)
	}
	if issue.Identifier != "ENG-9" {
		t.Fatalf("%+v", issue)
	}
	if sawAuth != "lin_api_test" {
		t.Fatalf("auth %q", sawAuth)
	}
	cmt, err := cfg.CreateComment(context.Background(), "ENG-9", "— Vibium (generated)")
	if err != nil {
		t.Fatal(err)
	}
	if cmt.ID != "c1" {
		t.Fatalf("%+v", cmt)
	}
}

func TestMissingKey(t *testing.T) {
	err := (Config{}).Validate()
	if err == nil || !strings.Contains(err.Error(), "setup tasks") {
		t.Fatalf("%v", err)
	}
}
