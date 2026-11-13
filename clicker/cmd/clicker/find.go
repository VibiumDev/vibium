package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibium/clicker/internal/api"
)

// findFlags are shared by find and its locator subcommands.
type findFlags struct {
	all      bool
	limit    int
	strategy string
	details  bool
}

func newFindCmd() *cobra.Command {
	var timeout time.Duration
	flags := &findFlags{}
	cmd := &cobra.Command{
		Use:   "find [description]",
		Short: "Find elements by description, reference image, or locator",
		Long: `Find elements and return @eN refs for later commands.

A bare string is a natural-language description. Use a subcommand for an
exact locator: css, tag, text, role, label, placeholder, testid, xpath,
alt, title, or image for a reference picture.

A description is matched against the page's names and roles first. When
that is not enough and an AI provider is configured (the same settings
as run and check), the screenshot is sent to a vision model. An image is
matched locally first; with a provider configured the screenshot and the
reference go to the model only if local matching finds nothing.`,
		Example: `  vibium find "the Add to Cart button"
  # → @e1 [button] "Add to Cart"

  vibium find "the main heading"
  vibium find "product thumbnail" --all

  vibium find css "button.primary"
  # Exact CSS (what a bare string meant before)

  vibium find image ./thumbnail.png --json
  # Where the picture appears, with bounding boxes

  vibium find "the red Save button" --strategy vlm --provider openai --model <vision-model>
  # Force the vision model for one call

  vibium find role button --strategy tree
  # Only the page's semantic tree, never a model`,
		Args: cobra.RangeArgs(0, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Error: requires a description, or a subcommand such as css, image, text, or role\n")
				os.Exit(1)
			}
			description := args[0]
			if len(args) == 2 && isURL(args[0]) {
				if _, err := daemonCall("browser_navigate", map[string]interface{}{"url": args[0]}); err != nil {
					printError(err)
					return
				}
				description = args[1]
			} else if len(args) == 2 {
				fmt.Fprintf(os.Stderr, "Error: find takes one description; quote it, or use a subcommand for a locator\n")
				os.Exit(1)
			}
			runFind(cmd, flags, timeout, map[string]interface{}{"description": description, "bare": true})
		},
	}
	addTimeoutFlag(cmd, &timeout)
	addModelFlags(cmd)
	cmd.PersistentFlags().BoolVar(&flags.all, "all", false, "Find all matching elements")
	cmd.PersistentFlags().IntVar(&flags.limit, "limit", 10, "Maximum number of elements to return (with --all)")
	cmd.PersistentFlags().StringVar(&flags.strategy, "strategy", "auto", "How to resolve: auto, cv (local image matching only), vlm (vision model only), or tree (page tree only)")
	cmd.PersistentFlags().BoolVar(&flags.details, "details", false, "Print diagnostics: method, model, raw scores")

	locator := func(name, short, example, key string) *cobra.Command {
		var t time.Duration
		sub := &cobra.Command{
			Use:     name + " [" + key + "]",
			Short:   short,
			Example: example,
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				runFind(cmd, flags, t, map[string]interface{}{key: args[0]})
			},
		}
		addTimeoutFlag(sub, &t)
		return sub
	}

	descriptionCmd := locator("description", "Find element by natural-language description",
		`  vibium find description "the Add to Cart button"
  # Same as a bare string; explicit when the text could read as a locator`, "description")
	addModelFlags(descriptionCmd)
	cmd.AddCommand(descriptionCmd)

	var imageTimeout time.Duration
	imageCmd := &cobra.Command{
		Use:   "image [path]",
		Short: "Find where a reference image appears on screen",
		Example: `  vibium find image ./thumbnail.png
  # → @e1 [img] "Product thumbnail"

  vibium find image ./thumbnail.png --all --json
  # Every occurrence, with bounding boxes

  vibium find image ./icon.png --strategy cv
  # Local feature matching only; nothing leaves the machine`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			data, err := readImageFile(args[0])
			if err != nil {
				printError(err)
				return
			}
			runFind(cmd, flags, imageTimeout, map[string]interface{}{"image": data})
		},
	}
	addTimeoutFlag(imageCmd, &imageTimeout)
	addModelFlags(imageCmd)
	cmd.AddCommand(imageCmd)

	cmd.AddCommand(locator("css", "Find element by CSS selector",
		`  vibium find css "button.primary"
  # → @e1 [button] "Submit"

  vibium find css "a" --all
  # → @e1 [a] "Home"  @e2 [a] "About"  ...`, "css"))
	cmd.AddCommand(locator("tag", "Find element by HTML tag name",
		`  vibium find tag h1
  # → @e1 [h1] "Example Domain"`, "tag"))
	cmd.AddCommand(locator("text", "Find element by text content",
		`  vibium find text "Sign In"
  # → @e1 [button] "Sign In"`, "text"))

	var roleTimeout time.Duration
	roleCmd := &cobra.Command{
		Use:   "role [role]",
		Short: "Find element by ARIA role",
		Example: `  vibium find role button
  # → @e1 [button] "Submit"

  vibium find role heading --name "Example"
  # Find heading with accessible name "Example"`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolArgs := map[string]interface{}{"role": args[0]}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				toolArgs["name"] = name
			}
			runFind(cmd, flags, roleTimeout, toolArgs)
		},
	}
	roleCmd.Flags().String("name", "", "Accessible name filter")
	addTimeoutFlag(roleCmd, &roleTimeout)
	cmd.AddCommand(roleCmd)

	cmd.AddCommand(locator("label", "Find input by associated label text",
		`  vibium find label "Email"
  # → @e1 [input type="email"] placeholder="Email"`, "label"))
	cmd.AddCommand(locator("placeholder", "Find element by placeholder attribute",
		`  vibium find placeholder "Search..."
  # → @e1 [input] placeholder="Search..."`, "placeholder"))
	cmd.AddCommand(locator("testid", "Find element by data-testid attribute",
		`  vibium find testid "submit-btn"
  # → @e1 [button] data-testid="submit-btn"`, "testid"))
	cmd.AddCommand(locator("xpath", "Find element by XPath expression",
		`  vibium find xpath "//div[@class='main']"
  # → @e1 [div.main] ...`, "xpath"))
	cmd.AddCommand(locator("alt", "Find element by alt attribute",
		`  vibium find alt "Logo"`, "alt"))
	cmd.AddCommand(locator("title", "Find element by title attribute",
		`  vibium find title "Close"`, "title"))

	return cmd
}

// runFind sends one find to the daemon with the shared flags applied.
func runFind(cmd *cobra.Command, flags *findFlags, timeout time.Duration, toolArgs map[string]interface{}) {
	toolArgs["timeout"] = float64(timeout.Milliseconds())
	toolArgs["strategy"] = flags.strategy
	if flags.details {
		toolArgs["details"] = true
	}
	if cmd.Flags().Lookup("provider") != nil {
		for name, value := range modelOverrideParams(cmd) {
			toolArgs[name] = value
		}
	}
	tool := "browser_find"
	if flags.all {
		tool = "browser_find_all"
		toolArgs["limit"] = float64(flags.limit)
	}
	result, err := daemonCall(tool, toolArgs)
	if err != nil {
		printError(err)
		return
	}
	printResult(result)
}

// modelOverrideParams maps the AI flags to the wire names check and run use.
func modelOverrideParams(cmd *cobra.Command) map[string]interface{} {
	out := map[string]interface{}{}
	for flag, name := range map[string]string{"provider": "provider", "model": "model", "ai-base-url": "aiBaseURL", "reasoning-effort": "reasoningEffort"} {
		if cmd.Flags().Changed(flag) {
			out[name], _ = cmd.Flags().GetString(flag)
		}
	}
	return out
}

// readImageFile reads a reference image for the daemon, which never opens
// files itself.
func readImageFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot read image %s: %w", path, err)
	}
	if info.Size() > api.MaxImageBytes {
		return "", fmt.Errorf("image %s is %d bytes; the limit is %d MB", path, info.Size(), api.MaxImageBytes/1024/1024)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read image %s: %w", path, err)
	}
	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") {
		return "", fmt.Errorf("image %s must be a PNG or JPEG", path)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// isURL returns true if the string looks like a URL (starts with http:// or https://).
func isURL(s string) bool {
	return len(s) > 8 && (s[:7] == "http://" || s[:8] == "https://")
}
