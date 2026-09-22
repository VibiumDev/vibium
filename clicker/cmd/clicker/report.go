package main

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/linear"
	"github.com/vibium/clicker/internal/verifier"
)

var issueRef = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "File a Run or Check result to a task manager",
		Long:  "Task managers are optional. Linear is the first one: signed issues with artifact filepaths.",
	}
	cmd.AddCommand(newReportLinearCmd())
	return cmd
}

func newReportLinearCmd() *cobra.Command {
	var commentOn, playbook, model string
	cmd := &cobra.Command{
		Use:   "linear <pack-dir>",
		Short: "File a Vibium pack directory as a signed Linear issue",
		Long: `Create a Linear issue from a pack directory (run.json, screenshots, recording zip).
The issue body ends with "Signed by Vibium" and absolute artifact paths.
Pass --comment ISSUE to add a signed comment on an existing issue instead.

Requires vibium setup tasks (VIBIUM_LINEAR_API_KEY and VIBIUM_LINEAR_TEAM).`,
		Example: `  vibium report linear ./out/pay
  # Creates an issue from out/pay/run.json, signed with paths in that directory.

  vibium report linear ./out/pay --comment ENG-12
  # Comments on ENG-12 instead of creating.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pack, err := linear.LoadPack(args[0])
			if err != nil {
				printError(err)
				return
			}
			pack.Version = version
			pack.Playbook = playbook
			pack.Model = model
			pack.Command = "vibium report linear " + args[0]
			posted, err := postLinearPack(cmd.Context(), pack, commentOn)
			if err != nil {
				printError(err)
				return
			}
			if jsonOutput {
				printJSON(jsonEnvelope{OK: true, Result: posted})
				return
			}
			fmt.Println(posted.URL)
		},
	}
	cmd.Flags().StringVar(&commentOn, "comment", "", "Existing issue identifier (ENG-12) to comment on instead of creating")
	cmd.Flags().StringVar(&playbook, "playbook", "", "Playbook name to record in the signature (inspect, walk)")
	cmd.Flags().StringVar(&model, "model", "", "Model id to record in the signature")
	return cmd
}

type linearPost struct {
	Provider   string `json:"provider"`
	Action     string `json:"action"`
	Identifier string `json:"identifier,omitempty"`
	URL        string `json:"url"`
}

func postLinearPack(ctx context.Context, pack linear.Pack, commentOn string) (*linearPost, error) {
	cfg := linear.FromEnv()
	if commentOn != "" {
		cmt, err := cfg.CreateComment(ctx, commentOn, pack.CommentBody())
		if err != nil {
			return nil, err
		}
		return &linearPost{Provider: "linear", Action: "comment", Identifier: commentOn, URL: cmt.URL}, nil
	}
	issue, err := cfg.CreateIssue(ctx, pack.Title(), pack.Description())
	if err != nil {
		return nil, err
	}
	return &linearPost{Provider: "linear", Action: "create", Identifier: issue.Identifier, URL: issue.URL}, nil
}

func parseReportFlag(flag string) (provider, issue string, err error) {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return "", "", nil
	}
	provider, rest, _ := strings.Cut(flag, ":")
	provider = strings.ToLower(provider)
	if provider != "linear" {
		return "", "", fmt.Errorf("unknown report target %q; only linear is wired", flag)
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "linear", "", nil
	}
	if !issueRef.MatchString(rest) {
		return "", "", fmt.Errorf("report linear:%s is not an issue id (expected ENG-123)", rest)
	}
	return "linear", rest, nil
}

func maybeReportLinear(ctx context.Context, reportFlag, output string, result interface {
	Verdict() string
	RecordingSummary() (string, string, []verifier.Evidence)
}, command, playbook, model, host string) (*linearPost, error) {
	provider, issue, err := parseReportFlag(reportFlag)
	if err != nil || provider == "" {
		return nil, err
	}
	_, summary, evidence := result.RecordingSummary()
	dir := ""
	if output != "" {
		dir = filepath.Dir(output)
	}
	pack := linear.Pack{
		Dir:      dir,
		Version:  version,
		Command:  command,
		Playbook: playbook,
		Model:    model,
		Result: linear.RunResult{
			Status:  strings.ToLower(result.Verdict()),
			Summary: summary,
			Host:    host,
		},
	}
	for _, e := range evidence {
		pack.Result.Evidence = append(pack.Result.Evidence, linear.EvidenceLine{Type: e.Type, Summary: e.Summary})
	}
	return postLinearPack(ctx, pack, issue)
}
