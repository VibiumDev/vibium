package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultEndpoint = "https://api.linear.app/graphql"

// Config is how Vibium talks to Linear. Keys never go in issue bodies.
type Config struct {
	APIKey   string
	TeamKey  string
	TeamID   string
	Endpoint string
	HTTP     *http.Client
}

// FromEnv reads VIBIUM_LINEAR_* and LINEAR_API_KEY.
func FromEnv() Config {
	key := strings.TrimSpace(os.Getenv("VIBIUM_LINEAR_API_KEY"))
	if key == "" {
		key = strings.TrimSpace(os.Getenv("LINEAR_API_KEY"))
	}
	ep := strings.TrimSpace(os.Getenv("VIBIUM_LINEAR_ENDPOINT"))
	if ep == "" {
		ep = defaultEndpoint
	}
	return Config{
		APIKey:   key,
		TeamKey:  strings.TrimSpace(os.Getenv("VIBIUM_LINEAR_TEAM")),
		TeamID:   strings.TrimSpace(os.Getenv("VIBIUM_LINEAR_TEAM_ID")),
		Endpoint: ep,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("Linear API key missing; run vibium setup tasks or set VIBIUM_LINEAR_API_KEY")
	}
	return nil
}

// Issue is a created Linear issue.
type Issue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

// Comment is a created Linear comment.
type Comment struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

func (c Config) do(ctx context.Context, query string, vars map[string]any, dest any) error {
	if err := c.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(gqlRequest{Query: query, Variables: vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.APIKey)
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	var parsed struct {
		Data   json.RawMessage `json:"data"`
		Errors []gqlError      `json:"errors"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("Linear returned non-JSON (HTTP %d)", res.StatusCode)
	}
	if len(parsed.Errors) > 0 {
		return fmt.Errorf("Linear: %s", parsed.Errors[0].Message)
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("Linear HTTP %d", res.StatusCode)
	}
	if dest != nil {
		if err := json.Unmarshal(parsed.Data, dest); err != nil {
			return fmt.Errorf("Linear response shape: %w", err)
		}
	}
	return nil
}

func (c Config) endpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	return defaultEndpoint
}

// ResolveTeamID returns TeamID, or looks up TeamKey.
func (c Config) ResolveTeamID(ctx context.Context) (string, error) {
	if c.TeamID != "" {
		return c.TeamID, nil
	}
	if c.TeamKey == "" {
		return "", fmt.Errorf("set VIBIUM_LINEAR_TEAM to a Linear team key (the prefix on ENG-123)")
	}
	var data struct {
		Teams struct {
			Nodes []struct {
				ID  string `json:"id"`
				Key string `json:"key"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	q := `query($key: String!) { teams(filter: { key: { eq: $key } }) { nodes { id key } } }`
	if err := c.do(ctx, q, map[string]any{"key": c.TeamKey}, &data); err != nil {
		return "", err
	}
	if len(data.Teams.Nodes) == 0 {
		return "", fmt.Errorf("Linear team %q not found", c.TeamKey)
	}
	return data.Teams.Nodes[0].ID, nil
}

// CreateIssue files an issue on the configured team.
func (c Config) CreateIssue(ctx context.Context, title, description string) (*Issue, error) {
	teamID, err := c.ResolveTeamID(ctx)
	if err != nil {
		return nil, err
	}
	var data struct {
		IssueCreate struct {
			Success bool  `json:"success"`
			Issue   Issue `json:"issue"`
		} `json:"issueCreate"`
	}
	q := `mutation($input: IssueCreateInput!) { issueCreate(input: $input) { success issue { id identifier url title } } }`
	if err := c.do(ctx, q, map[string]any{"input": map[string]any{
		"teamId":      teamID,
		"title":       title,
		"description": description,
	}}, &data); err != nil {
		return nil, err
	}
	if !data.IssueCreate.Success || data.IssueCreate.Issue.Identifier == "" {
		return nil, fmt.Errorf("Linear issueCreate did not succeed")
	}
	return &data.IssueCreate.Issue, nil
}

// CreateComment posts on an existing issue. issueRef may be an identifier (ENG-1) or UUID.
func (c Config) CreateComment(ctx context.Context, issueRef, body string) (*Comment, error) {
	var data struct {
		CommentCreate struct {
			Success bool    `json:"success"`
			Comment Comment `json:"comment"`
		} `json:"commentCreate"`
	}
	q := `mutation($input: CommentCreateInput!) { commentCreate(input: $input) { success comment { id url } } }`
	if err := c.do(ctx, q, map[string]any{"input": map[string]any{
		"issueId": issueRef,
		"body":    body,
	}}, &data); err != nil {
		return nil, err
	}
	if !data.CommentCreate.Success {
		return nil, fmt.Errorf("Linear commentCreate did not succeed")
	}
	return &data.CommentCreate.Comment, nil
}
