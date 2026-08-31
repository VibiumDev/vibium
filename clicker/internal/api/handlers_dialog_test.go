package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// recordingTransport captures messages sent to the client.
type recordingTransport struct {
	sent []string
}

func (t *recordingTransport) ID() uint64            { return 1 }
func (t *recordingTransport) Send(msg string) error { t.sent = append(t.sent, msg); return nil }
func (t *recordingTransport) Close() error          { return nil }

const promptOpened = `{"method":"browsingContext.userPromptOpened","params":{"context":"ctx-1","type":"alert","message":"hi"}}`

// The router owns the no-handler default: a prompt that opens while no client
// dialog handler is registered is the router's to dismiss (#446).
func TestAutoDismissContextDefaultsToDismiss(t *testing.T) {
	session := &BrowserSession{}

	if got := session.autoDismissContext(promptOpened); got != "ctx-1" {
		t.Fatalf("expected the prompt's context %q, got %q", "ctx-1", got)
	}
}

func TestAutoDismissContextIgnoresOtherEvents(t *testing.T) {
	session := &BrowserSession{}

	for _, msg := range []string{
		`{"method":"browsingContext.userPromptClosed","params":{"context":"ctx-1"}}`,
		`{"method":"log.entryAdded","params":{"context":"ctx-1"}}`,
		`{"id":5,"type":"success","result":{}}`,
		`not json`,
	} {
		if got := session.autoDismissContext(msg); got != "" {
			t.Fatalf("message %q should not be auto-dismissed, got context %q", msg, got)
		}
	}
}

// A page that registered a dialog handler flips its own context to manual,
// and the router keeps its hands off that context's prompts from then on.
func TestSetPolicyManualStopsAutoDismiss(t *testing.T) {
	router := &Router{}
	client := &recordingTransport{}
	session := &BrowserSession{Client: client}

	router.handleDialogSetPolicy(session, bidiCommand{
		ID:     7,
		Method: "vibium:dialog.setPolicy",
		Params: map[string]interface{}{"context": "ctx-1", "policy": "manual"},
	})

	if got := session.autoDismissContext(promptOpened); got != "" {
		t.Fatalf("manual policy must leave the prompt open, got context %q", got)
	}
	assertResponseType(t, client, 7, "success")

	// The policy is per context: another page's prompt is still the router's.
	other := `{"method":"browsingContext.userPromptOpened","params":{"context":"ctx-2","type":"confirm"}}`
	if got := session.autoDismissContext(other); got != "ctx-2" {
		t.Fatalf("a context without handlers keeps the default, got %q", got)
	}

	// The last handler deregistering hands the default back.
	router.handleDialogSetPolicy(session, bidiCommand{
		ID:     8,
		Method: "vibium:dialog.setPolicy",
		Params: map[string]interface{}{"context": "ctx-1", "policy": "dismiss"},
	})

	if got := session.autoDismissContext(promptOpened); got != "ctx-1" {
		t.Fatalf("dismiss policy must reclaim the prompt, got context %q", got)
	}
	assertResponseType(t, client, 8, "success")
}

func TestSetPolicyRequiresContext(t *testing.T) {
	router := &Router{}
	client := &recordingTransport{}
	session := &BrowserSession{Client: client}

	router.handleDialogSetPolicy(session, bidiCommand{
		ID:     10,
		Method: "vibium:dialog.setPolicy",
		Params: map[string]interface{}{"policy": "manual"},
	})

	assertResponseType(t, client, 10, "error")
	if got := session.autoDismissContext(promptOpened); got != "ctx-1" {
		t.Fatalf("a rejected policy must leave the default in place, got %q", got)
	}
}

func TestSetPolicyRejectsUnknownPolicy(t *testing.T) {
	router := &Router{}
	client := &recordingTransport{}
	session := &BrowserSession{Client: client}

	router.handleDialogSetPolicy(session, bidiCommand{
		ID:     9,
		Method: "vibium:dialog.setPolicy",
		Params: map[string]interface{}{"context": "ctx-1", "policy": "accept"},
	})

	assertResponseType(t, client, 9, "error")
	if !strings.Contains(client.sent[len(client.sent)-1], "invalid dialog policy") {
		t.Fatalf("error should name the bad policy, got %s", client.sent[len(client.sent)-1])
	}
	// A rejected policy must not change the default.
	if got := session.autoDismissContext(promptOpened); got != "ctx-1" {
		t.Fatalf("rejected policy must leave the default in place, got context %q", got)
	}
}

func assertResponseType(t *testing.T, client *recordingTransport, id int, wantType string) {
	t.Helper()
	if len(client.sent) == 0 {
		t.Fatal("no response sent to client")
	}
	var resp struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(client.sent[len(client.sent)-1]), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if resp.ID != id || resp.Type != wantType {
		t.Fatalf("want id=%d type=%s, got id=%d type=%s", id, wantType, resp.ID, resp.Type)
	}
}
