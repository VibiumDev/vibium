package mcp

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/features"
	"github.com/vibium/clicker/internal/log"
)

// Handlers manages browser session state and executes tool calls.
type Handlers struct {
	launchResult  *browser.LaunchResult
	client        *bidi.Client
	conn          *bidi.Connection
	screenshotDir string
	headless      bool
	refMap        map[string]string // @e1 -> CSS selector
	lastMap       string            // last map output (for diff)
	traceRecorder *traceRecorder
	downloadDir   string
}

// traceRecorder records browser traces (screenshots + snapshots).
type traceRecorder struct {
	name        string
	screenshots bool
	snapshots   bool
	startTime   time.Time
	done        chan struct{}
	mu          sync.Mutex
	screenData  []string // base64-encoded PNGs
	snapData    []string // HTML snapshots
}

func (t *traceRecorder) addScreenshot(data string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.screenData = append(t.screenData, data)
}

func (t *traceRecorder) addSnapshot(html string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.snapData = append(t.snapData, html)
}

func (t *traceRecorder) writeZip(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// Write screenshots
	for i, data := range t.screenData {
		fw, err := w.Create(fmt.Sprintf("screenshots/%04d.png", i))
		if err != nil {
			return err
		}
		pngData, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return err
		}
		if _, err := fw.Write(pngData); err != nil {
			return err
		}
	}

	// Write snapshots
	for i, html := range t.snapData {
		fw, err := w.Create(fmt.Sprintf("snapshots/%04d.html", i))
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(html)); err != nil {
			return err
		}
	}

	// Write metadata
	meta := map[string]interface{}{
		"name":        t.name,
		"startTime":   t.startTime.Format(time.RFC3339),
		"screenshots":  len(t.screenData),
		"snapshots":   len(t.snapData),
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	fw, err := w.Create("metadata.json")
	if err != nil {
		return err
	}
	_, err = fw.Write(metaJSON)
	return err
}

// NewHandlers creates a new Handlers instance.
// screenshotDir specifies where screenshots are saved. If empty, file saving is disabled.
// headless controls whether the browser is launched in headless mode.
func NewHandlers(screenshotDir string, headless bool) *Handlers {
	return &Handlers{
		screenshotDir: screenshotDir,
		headless:      headless,
	}
}

// Call executes a tool by name with the given arguments.
func (h *Handlers) Call(name string, args map[string]interface{}) (*ToolsCallResult, error) {
	log.Debug("tool call", "name", name, "args", args)

	switch name {
	case "browser_launch":
		return h.browserLaunch(args)
	case "browser_navigate":
		return h.browserNavigate(args)
	case "browser_click":
		return h.browserClick(args)
	case "browser_type":
		return h.browserType(args)
	case "browser_screenshot":
		return h.browserScreenshot(args)
	case "browser_find":
		return h.browserFind(args)
	case "browser_evaluate":
		return h.browserEvaluate(args)
	case "browser_quit":
		return h.browserQuit(args)
	case "browser_get_text":
		return h.browserGetText(args)
	case "browser_get_url":
		return h.browserGetURL(args)
	case "browser_get_title":
		return h.browserGetTitle(args)
	case "browser_get_html":
		return h.browserGetHTML(args)
	case "browser_find_all":
		return h.browserFindAll(args)
	case "browser_wait":
		return h.browserWait(args)
	case "browser_hover":
		return h.browserHover(args)
	case "browser_select":
		return h.browserSelect(args)
	case "browser_scroll":
		return h.browserScroll(args)
	case "browser_keys":
		return h.browserKeys(args)
	case "browser_new_tab":
		return h.browserNewTab(args)
	case "browser_list_tabs":
		return h.browserListTabs(args)
	case "browser_switch_tab":
		return h.browserSwitchTab(args)
	case "browser_close_tab":
		return h.browserCloseTab(args)
	case "browser_a11y_tree":
		return h.browserA11yTree(args)
	case "page_clock_install":
		return h.pageClockInstall(args)
	case "page_clock_fast_forward":
		return h.pageClockFastForward(args)
	case "page_clock_run_for":
		return h.pageClockRunFor(args)
	case "page_clock_pause_at":
		return h.pageClockPauseAt(args)
	case "page_clock_resume":
		return h.pageClockResume(args)
	case "page_clock_set_fixed_time":
		return h.pageClockSetFixedTime(args)
	case "page_clock_set_system_time":
		return h.pageClockSetSystemTime(args)
	case "page_clock_set_timezone":
		return h.pageClockSetTimezone(args)
	case "browser_fill":
		return h.browserFill(args)
	case "browser_press":
		return h.browserPress(args)
	case "browser_back":
		return h.browserBack(args)
	case "browser_forward":
		return h.browserForward(args)
	case "browser_reload":
		return h.browserReload(args)
	case "browser_get_value":
		return h.browserGetValue(args)
	case "browser_get_attribute":
		return h.browserGetAttribute(args)
	case "browser_is_visible":
		return h.browserIsVisible(args)
	case "browser_check":
		return h.browserCheck(args)
	case "browser_uncheck":
		return h.browserUncheck(args)
	case "browser_scroll_into_view":
		return h.browserScrollIntoView(args)
	case "browser_wait_for_url":
		return h.browserWaitForURL(args)
	case "browser_wait_for_load":
		return h.browserWaitForLoad(args)
	case "browser_sleep":
		return h.browserSleep(args)
	case "browser_map":
		return h.browserMap(args)
	case "browser_diff_map":
		return h.browserDiffMap(args)
	case "browser_pdf":
		return h.browserPDF(args)
	case "browser_highlight":
		return h.browserHighlight(args)
	case "browser_dblclick":
		return h.browserDblClick(args)
	case "browser_focus":
		return h.browserFocus(args)
	case "browser_count":
		return h.browserCount(args)
	case "browser_is_enabled":
		return h.browserIsEnabled(args)
	case "browser_is_checked":
		return h.browserIsChecked(args)
	case "browser_wait_for_text":
		return h.browserWaitForText(args)
	case "browser_wait_for_fn":
		return h.browserWaitForFn(args)
	case "browser_dialog_accept":
		return h.browserDialogAccept(args)
	case "browser_dialog_dismiss":
		return h.browserDialogDismiss(args)
	case "browser_get_cookies":
		return h.browserGetCookies(args)
	case "browser_set_cookie":
		return h.browserSetCookie(args)
	case "browser_delete_cookies":
		return h.browserDeleteCookies(args)
	case "browser_mouse_move":
		return h.browserMouseMove(args)
	case "browser_mouse_down":
		return h.browserMouseDown(args)
	case "browser_mouse_up":
		return h.browserMouseUp(args)
	case "browser_mouse_click":
		return h.browserMouseClick(args)
	case "browser_drag":
		return h.browserDrag(args)
	case "browser_set_viewport":
		return h.browserSetViewport(args)
	case "browser_get_viewport":
		return h.browserGetViewport(args)
	case "browser_get_window":
		return h.browserGetWindow(args)
	case "browser_set_window":
		return h.browserSetWindow(args)
	case "browser_emulate_media":
		return h.browserEmulateMedia(args)
	case "browser_set_geolocation":
		return h.browserSetGeolocation(args)
	case "browser_set_content":
		return h.browserSetContent(args)
	case "browser_frames":
		return h.browserFrames(args)
	case "browser_frame":
		return h.browserFrame(args)
	case "browser_upload":
		return h.browserUpload(args)
	case "browser_trace_start":
		return h.browserTraceStart(args)
	case "browser_trace_stop":
		return h.browserTraceStop(args)
	case "browser_storage_state":
		return h.browserStorageState(args)
	case "browser_restore_storage":
		return h.browserRestoreStorage(args)
	case "browser_download_set_dir":
		return h.browserDownloadSetDir(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// Close cleans up any active browser sessions.
func (h *Handlers) Close() {
	if h.conn != nil {
		h.conn.Close()
		h.conn = nil
	}
	if h.launchResult != nil {
		h.launchResult.Close()
		h.launchResult = nil
	}
	h.client = nil
}

// browserLaunch launches a new browser session.
func (h *Handlers) browserLaunch(args map[string]interface{}) (*ToolsCallResult, error) {
	// If browser is already running, return success (no-op)
	if h.client != nil {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "Browser already running",
			}},
		}, nil
	}

	// Parse options — per-call headless overrides the default
	useHeadless := h.headless
	if val, ok := args["headless"].(bool); ok {
		useHeadless = val
	}

	// Launch browser
	launchResult, err := browser.Launch(browser.LaunchOptions{Headless: useHeadless})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to BiDi
	conn, err := bidi.Connect(launchResult.WebSocketURL)
	if err != nil {
		launchResult.Close()
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	h.launchResult = launchResult
	h.conn = conn
	h.client = bidi.NewClient(conn)

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Browser launched (headless: %v)", useHeadless),
		}},
	}, nil
}

// browserNavigate navigates to a URL.
func (h *Handlers) browserNavigate(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	result, err := h.client.Navigate("", url)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Navigated to %s", result.URL),
		}},
	}, nil
}

// browserClick clicks an element.
func (h *Handlers) browserClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	// Wait for element to be actionable
	opts := features.DefaultWaitOptions()
	if err := features.WaitForClick(h.client, "", selector, opts); err != nil {
		return nil, err
	}

	// Click the element
	if err := h.client.ClickElement("", selector); err != nil {
		return nil, fmt.Errorf("failed to click: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Clicked element: %s", selector),
		}},
	}, nil
}

// browserType types text into an element.
func (h *Handlers) browserType(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required")
	}

	// Wait for element to be actionable
	opts := features.DefaultWaitOptions()
	if err := features.WaitForType(h.client, "", selector, opts); err != nil {
		return nil, err
	}

	// Type into the element
	if err := h.client.TypeIntoElement("", selector, text); err != nil {
		return nil, fmt.Errorf("failed to type: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Typed into element: %s", selector),
		}},
	}, nil
}

// browserScreenshot captures a screenshot.
func (h *Handlers) browserScreenshot(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	fullPage, _ := args["fullPage"].(bool)
	annotate, _ := args["annotate"].(bool)

	// If annotate, run map first to get refs, then inject matching labels
	if annotate {
		if _, err := h.browserMap(map[string]interface{}{}); err != nil {
			return nil, fmt.Errorf("failed to map for annotation: %w", err)
		}

		// Build ordered list of selectors from refMap (@e1, @e2, ...)
		selectors := make([]string, 0, len(h.refMap))
		for i := 1; i <= len(h.refMap); i++ {
			ref := fmt.Sprintf("@e%d", i)
			if sel, ok := h.refMap[ref]; ok {
				selectors = append(selectors, sel)
			}
		}

		annotateScript := `(selectors) => {
			let count = 0;
			for (let i = 0; i < selectors.length; i++) {
				const el = document.querySelector(selectors[i]);
				if (!el) continue;
				const rect = el.getBoundingClientRect();
				if (rect.width === 0 || rect.height === 0) continue;
				const label = document.createElement('div');
				label.className = '__vibium_annotation';
				label.textContent = i + 1;
				label.style.cssText = 'position:fixed;z-index:2147483647;background:red;color:white;font:bold 11px sans-serif;padding:1px 4px;border-radius:8px;pointer-events:none;line-height:16px;min-width:16px;text-align:center;left:' + (rect.left - 2) + 'px;top:' + (rect.top - 2) + 'px;';
				document.body.appendChild(label);
				count++;
			}
			return JSON.stringify({count: count});
		}`
		if _, err := h.client.CallFunction("", annotateScript, []interface{}{selectors}); err != nil {
			return nil, fmt.Errorf("failed to annotate: %w", err)
		}
	}

	var base64Data string
	var err error
	if fullPage {
		base64Data, err = h.client.CaptureFullPageScreenshot("")
	} else {
		base64Data, err = h.client.CaptureScreenshot("")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Clean up annotation labels
	if annotate {
		cleanupScript := `() => {
			document.querySelectorAll('.__vibium_annotation').forEach(el => el.remove());
			return 'cleaned';
		}`
		h.client.CallFunction("", cleanupScript, nil)
	}

	// If filename provided, save to file (only if screenshotDir is configured)
	if filename, ok := args["filename"].(string); ok && filename != "" {
		if h.screenshotDir == "" {
			return nil, fmt.Errorf("screenshot file saving is disabled (use --screenshot-dir to enable)")
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(h.screenshotDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create screenshot directory: %w", err)
		}

		// Use only the basename to prevent path traversal
		safeName := filepath.Base(filename)
		fullPath := filepath.Join(h.screenshotDir, safeName)

		pngData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %w", err)
		}
		if err := os.WriteFile(fullPath, pngData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save screenshot: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Screenshot saved to %s", fullPath),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type:     "image",
			Data:     base64Data,
			MimeType: "image/png",
		}},
	}, nil
}

// browserFind finds an element and returns its info.
// Supports CSS selector or semantic locators (text, label, placeholder, testid, xpath, alt, title).
func (h *Handlers) browserFind(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	// Check for semantic locators
	role, _ := args["role"].(string)
	text, _ := args["text"].(string)
	label, _ := args["label"].(string)
	placeholder, _ := args["placeholder"].(string)
	testid, _ := args["testid"].(string)
	xpath, _ := args["xpath"].(string)
	alt, _ := args["alt"].(string)
	title, _ := args["title"].(string)

	hasSemantic := role != "" || text != "" || label != "" || placeholder != "" || testid != "" || xpath != "" || alt != "" || title != ""

	if hasSemantic {
		timeout := features.DefaultTimeout
		if t, ok := args["timeout"].(float64); ok {
			timeout = time.Duration(t) * time.Millisecond
		}

		script := findBySemanticScript()
		result, err := pollCallFunction(h, script, []interface{}{role, text, label, placeholder, testid, xpath, alt, title}, timeout)
		if err != nil {
			desc := ""
			for _, pair := range []struct{ k, v string }{
				{"role", role}, {"text", text}, {"label", label}, {"placeholder", placeholder},
				{"testid", testid}, {"xpath", xpath}, {"alt", alt}, {"title", title},
			} {
				if pair.v != "" {
					if desc != "" {
						desc += ", "
					}
					desc += pair.k + "=" + pair.v
				}
			}
			return nil, fmt.Errorf("element not found: %s (timeout %s)", desc, timeout)
		}

		// Parse JSON result
		var found struct {
			Selector string `json:"selector"`
			Label    string `json:"label"`
		}
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", result)), &found); err != nil {
			return nil, fmt.Errorf("failed to parse find result: %w", err)
		}

		// Store ref in refMap
		h.refMap = make(map[string]string)
		h.refMap["@e1"] = found.Selector

		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("@e1 %s", found.Label),
			}},
		}, nil
	}

	// CSS selector mode
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector or semantic locator (role, text, label, placeholder, testid, xpath, alt, title) is required")
	}
	selector = h.resolveSelector(selector)

	// Run getLabel in browser to get consistent label format
	labelScript := `(selector) => {
		` + GetLabelJS() + `
		const el = document.querySelector(selector);
		if (!el) return null;
		return getLabel(el);
	}`
	labelResult, err := h.client.CallFunction("", labelScript, []interface{}{selector})
	if err != nil {
		return nil, err
	}

	// Store ref in refMap
	h.refMap = make(map[string]string)
	h.refMap["@e1"] = selector

	labelStr := fmt.Sprintf("%v", labelResult)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("@e1 %s", labelStr),
		}},
	}, nil
}

// findBySemanticScript returns the JS function for finding elements by semantic criteria.
// Returns JSON: {"selector":"...","label":"...","tag":"...","text":"...","box":{...}}
func findBySemanticScript() string {
	return `(role, text, label, placeholder, testid, xpath, alt, title) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `

		const IMPLICIT_ROLES = {
			A: (el) => el.hasAttribute('href') ? 'link' : '',
			AREA: (el) => el.hasAttribute('href') ? 'link' : '',
			ARTICLE: () => 'article',
			ASIDE: () => 'complementary',
			BUTTON: () => 'button',
			DETAILS: () => 'group',
			DIALOG: () => 'dialog',
			FOOTER: () => 'contentinfo',
			FORM: () => 'form',
			H1: () => 'heading', H2: () => 'heading', H3: () => 'heading',
			H4: () => 'heading', H5: () => 'heading', H6: () => 'heading',
			HEADER: () => 'banner',
			HR: () => 'separator',
			IMG: (el) => el.getAttribute('alt') ? 'img' : 'presentation',
			INPUT: (el) => {
				const t = (el.getAttribute('type') || 'text').toLowerCase();
				const map = {button:'button',checkbox:'checkbox',image:'button',
					number:'spinbutton',radio:'radio',range:'slider',
					reset:'button',search:'searchbox',submit:'button',text:'textbox',
					email:'textbox',tel:'textbox',url:'textbox',password:'textbox'};
				return map[t] || 'textbox';
			},
			LI: () => 'listitem',
			MAIN: () => 'main',
			MENU: () => 'list',
			NAV: () => 'navigation',
			OL: () => 'list',
			OPTION: () => 'option',
			OUTPUT: () => 'status',
			PROGRESS: () => 'progressbar',
			SECTION: () => 'region',
			SELECT: (el) => el.hasAttribute('multiple') ? 'listbox' : 'combobox',
			SUMMARY: () => 'button',
			TABLE: () => 'table',
			TBODY: () => 'rowgroup', THEAD: () => 'rowgroup', TFOOT: () => 'rowgroup',
			TD: () => 'cell',
			TEXTAREA: () => 'textbox',
			TH: () => 'columnheader',
			TR: () => 'row',
			UL: () => 'list',
		};

		function getImplicitRole(el) {
			const explicit = el.getAttribute('role');
			if (explicit) return explicit.toLowerCase();
			const fn = IMPLICIT_ROLES[el.tagName];
			return fn ? fn(el).toLowerCase() : '';
		}

		function getName(el) {
			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return ariaLabel;
			const labelledBy = el.getAttribute('aria-labelledby');
			if (labelledBy) {
				const parts = labelledBy.split(/\s+/).map(id => {
					const ref = document.getElementById(id);
					return ref ? (ref.textContent || '').trim() : '';
				}).filter(Boolean);
				if (parts.length) return parts.join(' ');
			}
			if (el.id) {
				const assocLabel = document.querySelector('label[for="' + el.id + '"]');
				if (assocLabel) return (assocLabel.textContent || '').trim();
			}
			const ph = el.getAttribute('placeholder');
			if (ph) return ph;
			const altAttr = el.getAttribute('alt');
			if (altAttr) return altAttr;
			const titleAttr = el.getAttribute('title');
			if (titleAttr) return titleAttr;
			return (el.textContent || '').trim();
		}

		let el = null;

		if (role) {
			// Role-based matching: walk all elements, filter by role + other criteria
			const roleLower = role.toLowerCase();
			const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT);
			const found = [];
			let node;
			while (node = walker.nextNode()) {
				if (getImplicitRole(node) !== roleLower) continue;
				// Apply additional filters
				if (text && !(node.textContent || '').trim().includes(text)) continue;
				if (label) {
					const elName = getName(node);
					if (!elName.includes(label)) continue;
				}
				if (placeholder) {
					const ph = node.getAttribute('placeholder');
					if (!ph || !ph.includes(placeholder)) continue;
				}
				if (testid) {
					const tid = node.getAttribute('data-testid');
					if (tid !== testid) continue;
				}
				if (alt) {
					const a = node.getAttribute('alt');
					if (!a || !a.includes(alt)) continue;
				}
				if (title) {
					const t = node.getAttribute('title');
					if (!t || !t.includes(title)) continue;
				}
				found.push(node);
			}
			if (found.length === 0) return null;
			// Pick best: prefer shortest text match if text filter is used
			el = found[0];
			if (text && found.length > 1) {
				let bestLen = (el.textContent || '').length;
				for (let i = 1; i < found.length; i++) {
					const len = (found[i].textContent || '').length;
					if (len < bestLen) { el = found[i]; bestLen = len; }
				}
			}
		} else if (xpath) {
			const xresult = document.evaluate(xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
			el = xresult.singleNodeValue;
		} else if (testid) {
			el = document.querySelector('[data-testid="' + testid.replace(/"/g, '\\"') + '"]');
		} else if (placeholder) {
			el = document.querySelector('[placeholder="' + placeholder.replace(/"/g, '\\"') + '"]');
		} else if (alt) {
			el = document.querySelector('[alt="' + alt.replace(/"/g, '\\"') + '"]');
		} else if (title) {
			el = document.querySelector('[title="' + title.replace(/"/g, '\\"') + '"]');
		} else if (label) {
			// Try <label> with for= attribute pointing to an input
			const labels = document.querySelectorAll('label');
			for (const lbl of labels) {
				if (lbl.textContent.trim().includes(label)) {
					if (lbl.htmlFor) {
						el = document.getElementById(lbl.htmlFor);
					} else {
						el = lbl.querySelector('input, textarea, select');
					}
					if (el) break;
				}
			}
			// Fallback: aria-label
			if (!el) {
				el = document.querySelector('[aria-label="' + label.replace(/"/g, '\\"') + '"]');
			}
			// Fallback: aria-labelledby
			if (!el) {
				const all = document.querySelectorAll('[aria-labelledby]');
				for (const candidate of all) {
					const labelId = candidate.getAttribute('aria-labelledby');
					const labelEl = document.getElementById(labelId);
					if (labelEl && labelEl.textContent.trim().includes(label)) {
						el = candidate;
						break;
					}
				}
			}
		} else if (text) {
			// Find leaf elements containing the text
			const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT, {
				acceptNode: (node) => {
					if (node.offsetWidth === 0 && node.offsetHeight === 0) return NodeFilter.FILTER_REJECT;
					const style = window.getComputedStyle(node);
					if (style.display === 'none' || style.visibility === 'hidden') return NodeFilter.FILTER_REJECT;
					return NodeFilter.FILTER_ACCEPT;
				}
			});
			let best = null;
			let bestLen = Infinity;
			let node;
			while (node = walker.nextNode()) {
				const content = node.textContent.trim();
				if (content.includes(text) && content.length < bestLen) {
					// Prefer the most specific (smallest text) match
					best = node;
					bestLen = content.length;
				}
			}
			el = best;
		}

		if (!el) return null;

		const rect = el.getBoundingClientRect();
		return JSON.stringify({
			selector: getSelector(el),
			label: getLabel(el),
			tag: el.tagName.toLowerCase(),
			text: (el.textContent || '').trim().substring(0, 100),
			box: { x: Math.round(rect.x), y: Math.round(rect.y), w: Math.round(rect.width), h: Math.round(rect.height) }
		});
	}`
}

// browserEvaluate executes JavaScript code in the browser.
func (h *Handlers) browserEvaluate(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	result, err := h.client.Evaluate("", expression)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate: %w", err)
	}

	// Format result as string
	var resultText string
	switch v := result.(type) {
	case string:
		resultText = v
	case nil:
		resultText = "null"
	default:
		resultText = fmt.Sprintf("%v", v)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: resultText,
		}},
	}, nil
}

// browserQuit closes the browser session.
func (h *Handlers) browserQuit(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.launchResult == nil {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No browser session to close",
			}},
		}, nil
	}

	h.Close()

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Browser session closed",
		}},
	}, nil
}

// browserNewTab creates a new browser tab.
func (h *Handlers) browserNewTab(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	url, _ := args["url"].(string)

	contextID, err := h.client.CreateTab(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create tab: %w", err)
	}

	msg := "New tab opened"
	if url != "" {
		msg = fmt.Sprintf("New tab opened and navigated to %s", url)
	}
	_ = contextID

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserListTabs lists all open browser tabs.
func (h *Handlers) browserListTabs(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	tree, err := h.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tabs: %w", err)
	}

	var text string
	for i, ctx := range tree.Contexts {
		text += fmt.Sprintf("[%d] %s\n", i, ctx.URL)
	}
	if text == "" {
		text = "No tabs open"
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserSwitchTab switches to a tab by index or URL substring.
func (h *Handlers) browserSwitchTab(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	tree, err := h.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tabs: %w", err)
	}

	var contextID string

	// Try index first
	if idx, ok := args["index"].(float64); ok {
		i := int(idx)
		if i < 0 || i >= len(tree.Contexts) {
			return nil, fmt.Errorf("tab index %d out of range (0-%d)", i, len(tree.Contexts)-1)
		}
		contextID = tree.Contexts[i].Context
	} else if url, ok := args["url"].(string); ok && url != "" {
		// Search by URL substring
		for _, ctx := range tree.Contexts {
			if containsSubstring(ctx.URL, url) {
				contextID = ctx.Context
				break
			}
		}
		if contextID == "" {
			return nil, fmt.Errorf("no tab matching URL %q", url)
		}
	} else {
		return nil, fmt.Errorf("index or url is required")
	}

	if err := h.client.ActivateTab(contextID); err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Switched to tab: %s", contextID),
		}},
	}, nil
}

// browserCloseTab closes a tab by index (default: current tab).
func (h *Handlers) browserCloseTab(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	tree, err := h.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tabs: %w", err)
	}

	if len(tree.Contexts) == 0 {
		return nil, fmt.Errorf("no tabs open")
	}

	idx := 0
	if i, ok := args["index"].(float64); ok {
		idx = int(i)
	}

	if idx < 0 || idx >= len(tree.Contexts) {
		return nil, fmt.Errorf("tab index %d out of range (0-%d)", idx, len(tree.Contexts)-1)
	}

	contextID := tree.Contexts[idx].Context
	if err := h.client.CloseTab(contextID); err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Closed tab %d", idx),
		}},
	}, nil
}

// browserA11yTree returns the accessibility tree of the current page.
func (h *Handlers) browserA11yTree(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	interestingOnly := true
	if val, ok := args["everything"].(bool); ok {
		interestingOnly = !val
	}

	script := a11yTreeMCPScript()
	result, err := h.client.CallFunction("", script, []interface{}{interestingOnly})
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", result),
		}},
	}, nil
}

// a11yTreeMCPScript returns the JS function for the MCP a11y tree tool.
func a11yTreeMCPScript() string {
	return `(interestingOnly) => {
		const IMPLICIT_ROLES = {
			A: (el) => el.hasAttribute('href') ? 'link' : '',
			AREA: (el) => el.hasAttribute('href') ? 'link' : '',
			ARTICLE: () => 'article',
			ASIDE: () => 'complementary',
			BUTTON: () => 'button',
			DETAILS: () => 'group',
			DIALOG: () => 'dialog',
			FOOTER: () => 'contentinfo',
			FORM: () => 'form',
			H1: () => 'heading', H2: () => 'heading', H3: () => 'heading',
			H4: () => 'heading', H5: () => 'heading', H6: () => 'heading',
			HEADER: () => 'banner',
			HR: () => 'separator',
			IMG: (el) => el.getAttribute('alt') ? 'img' : 'presentation',
			INPUT: (el) => {
				const t = (el.getAttribute('type') || 'text').toLowerCase();
				const map = {button:'button',checkbox:'checkbox',image:'button',
					number:'spinbutton',radio:'radio',range:'slider',
					reset:'button',search:'searchbox',submit:'button',text:'textbox',
					email:'textbox',tel:'textbox',url:'textbox',password:'textbox'};
				return map[t] || 'textbox';
			},
			LI: () => 'listitem',
			MAIN: () => 'main',
			MENU: () => 'list',
			NAV: () => 'navigation',
			OL: () => 'list',
			OPTION: () => 'option',
			OUTPUT: () => 'status',
			PROGRESS: () => 'progressbar',
			SECTION: () => 'region',
			SELECT: (el) => el.hasAttribute('multiple') ? 'listbox' : 'combobox',
			SUMMARY: () => 'button',
			TABLE: () => 'table',
			TBODY: () => 'rowgroup', THEAD: () => 'rowgroup', TFOOT: () => 'rowgroup',
			TD: () => 'cell',
			TEXTAREA: () => 'textbox',
			TH: () => 'columnheader',
			TR: () => 'row',
			UL: () => 'list',
		};

		function getRole(el) {
			if (typeof el.computedRole === 'string' && el.computedRole !== '') return el.computedRole;
			const explicit = el.getAttribute('role');
			if (explicit) return explicit.toLowerCase();
			const fn = IMPLICIT_ROLES[el.tagName];
			return fn ? fn(el) : 'generic';
		}

		function getName(el) {
			if (typeof el.computedName === 'string') return el.computedName;
			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return ariaLabel;
			const labelledBy = el.getAttribute('aria-labelledby');
			if (labelledBy) {
				const parts = labelledBy.split(/\\s+/).map(id => {
					const ref = document.getElementById(id);
					return ref ? (ref.textContent || '').trim() : '';
				}).filter(Boolean);
				if (parts.length) return parts.join(' ');
			}
			if (el.id) {
				const assocLabel = document.querySelector('label[for="' + el.id + '"]');
				if (assocLabel) return (assocLabel.textContent || '').trim();
			}
			const placeholder = el.getAttribute('placeholder');
			if (placeholder) return placeholder;
			const alt = el.getAttribute('alt');
			if (alt) return alt;
			const title = el.getAttribute('title');
			if (title) return title;
			return '';
		}

		function getChildren(el) {
			if (el.shadowRoot) return Array.from(el.shadowRoot.children);
			return Array.from(el.children);
		}

		function getHeadingLevel(el) {
			const tag = el.tagName;
			if (tag === 'H1') return 1;
			if (tag === 'H2') return 2;
			if (tag === 'H3') return 3;
			if (tag === 'H4') return 4;
			if (tag === 'H5') return 5;
			if (tag === 'H6') return 6;
			const level = el.getAttribute('aria-level');
			if (level) return parseInt(level, 10);
			return undefined;
		}

		function buildNode(el) {
			const role = getRole(el);
			const name = getName(el);
			const childNodes = [];
			for (const child of getChildren(el)) {
				if (child.nodeType !== 1) continue;
				const nodes = buildNode(child);
				if (nodes) {
					if (Array.isArray(nodes)) childNodes.push(...nodes);
					else childNodes.push(nodes);
				}
			}
			if (interestingOnly) {
				if (role === 'none' || role === 'presentation') return childNodes.length ? childNodes : null;
				if (role === 'generic' && !name) return childNodes.length ? childNodes : null;
			}
			const node = { role: role };
			if (name) node.name = name;
			if (el.hasAttribute('disabled') || el.disabled) node.disabled = true;
			if (el.hasAttribute('aria-expanded')) node.expanded = el.getAttribute('aria-expanded') === 'true';
			if (document.activeElement === el) node.focused = true;
			if (typeof el.checked === 'boolean' && (el.type === 'checkbox' || el.type === 'radio')) {
				node.checked = el.checked;
			} else if (el.hasAttribute('aria-checked')) {
				const v = el.getAttribute('aria-checked');
				node.checked = v === 'true' ? true : v === 'mixed' ? 'mixed' : false;
			}
			if (el.hasAttribute('aria-pressed')) {
				const v = el.getAttribute('aria-pressed');
				node.pressed = v === 'true' ? true : v === 'mixed' ? 'mixed' : false;
			}
			if (el.hasAttribute('aria-selected') && el.getAttribute('aria-selected') === 'true') node.selected = true;
			if (el.hasAttribute('required') || el.required) node.required = true;
			if (el.hasAttribute('readonly') || el.readOnly) node.readonly = true;
			const level = getHeadingLevel(el);
			if (level !== undefined) node.level = level;
			if (childNodes.length) node.children = childNodes;
			return node;
		}

		const children = [];
		for (const child of getChildren(document.body)) {
			if (child.nodeType !== 1) continue;
			const nodes = buildNode(child);
			if (nodes) {
				if (Array.isArray(nodes)) children.push(...nodes);
				else children.push(nodes);
			}
		}
		return JSON.stringify({ role: 'WebArea', name: document.title, children: children });
	}`
}

// containsSubstring checks if s contains substr (case-sensitive).
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}

// browserHover moves the mouse over an element.
func (h *Handlers) browserHover(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	info, err := h.client.FindElement("", selector)
	if err != nil {
		return nil, err
	}

	x, y := info.GetCenter()
	if err := h.client.MoveMouse("", x, y); err != nil {
		return nil, fmt.Errorf("failed to hover: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Hovered over element: %s", selector),
		}},
	}, nil
}

// browserSelect selects an option in a <select> element.
func (h *Handlers) browserSelect(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	value, ok := args["value"].(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("value is required")
	}

	script := `(selector, value) => {
		const el = document.querySelector(selector);
		if (!el) return JSON.stringify({error: 'Element not found'});
		if (el.tagName.toLowerCase() !== 'select') return JSON.stringify({error: 'Element is not a <select>'});
		el.value = value;
		el.dispatchEvent(new Event('change', {bubbles: true}));
		return JSON.stringify({selected: el.value});
	}`

	result, err := h.client.CallFunction("", script, []interface{}{selector, value})
	if err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Selected value %q in %s (result: %v)", value, selector, result),
		}},
	}, nil
}

// browserScroll scrolls the page or an element.
func (h *Handlers) browserScroll(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	direction := "down"
	if d, ok := args["direction"].(string); ok && d != "" {
		direction = d
	}

	amount := 3
	if a, ok := args["amount"].(float64); ok {
		amount = int(a)
	}

	// Determine scroll target coordinates
	x, y := 0, 0
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		info, err := h.client.FindElement("", selector)
		if err != nil {
			return nil, err
		}
		cx, cy := info.GetCenter()
		x, y = int(cx), int(cy)
	} else {
		// Viewport center
		result, err := h.client.Evaluate("", "JSON.stringify({w: window.innerWidth, h: window.innerHeight})")
		if err == nil && result != nil {
			x, y = 400, 300 // Reasonable fallback
		}
	}

	// Map direction to deltas (120 pixels per scroll "notch")
	deltaX, deltaY := 0, 0
	pixels := amount * 120
	switch direction {
	case "down":
		deltaY = pixels
	case "up":
		deltaY = -pixels
	case "right":
		deltaX = pixels
	case "left":
		deltaX = -pixels
	default:
		return nil, fmt.Errorf("invalid direction: %q (use up, down, left, right)", direction)
	}

	if err := h.client.ScrollWheel("", x, y, deltaX, deltaY); err != nil {
		return nil, fmt.Errorf("failed to scroll: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Scrolled %s by %d", direction, amount),
		}},
	}, nil
}

// browserKeys presses a key or key combination.
func (h *Handlers) browserKeys(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	keys, ok := args["keys"].(string)
	if !ok || keys == "" {
		return nil, fmt.Errorf("keys is required")
	}

	if err := h.client.PressKeyCombo("", keys); err != nil {
		return nil, fmt.Errorf("failed to press keys: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Pressed keys: %s", keys),
		}},
	}, nil
}

// browserGetHTML returns the HTML content of the page or an element.
func (h *Handlers) browserGetHTML(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	outer, _ := args["outer"].(bool)

	var expr string
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		if outer {
			expr = fmt.Sprintf(`document.querySelector(%q)?.outerHTML || ''`, selector)
		} else {
			expr = fmt.Sprintf(`document.querySelector(%q)?.innerHTML || ''`, selector)
		}
	} else {
		expr = `document.documentElement.outerHTML`
	}

	result, err := h.client.Evaluate("", expr)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	html := ""
	if result != nil {
		html = fmt.Sprintf("%v", result)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: html,
		}},
	}, nil
}

// browserFindAll finds all elements matching a CSS selector.
func (h *Handlers) browserFindAll(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Use JS to find elements and generate selectors + labels
	findAllScript := `(selector, limit) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `
		const els = document.querySelectorAll(selector);
		const results = [];
		const n = Math.min(els.length, limit);
		for (let i = 0; i < n; i++) {
			const el = els[i];
			results.push({ selector: getSelector(el), label: getLabel(el) });
		}
		return JSON.stringify(results);
	}`
	result, err := h.client.CallFunction("", findAllScript, []interface{}{selector, limit})
	if err != nil {
		return nil, fmt.Errorf("failed to find elements: %w", err)
	}

	var elements []struct {
		Selector string `json:"selector"`
		Label    string `json:"label"`
	}
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", result)), &elements); err != nil {
		return nil, fmt.Errorf("failed to parse find-all results: %w", err)
	}

	// Build ref map and output
	h.refMap = make(map[string]string)
	var lines []string
	for i, el := range elements {
		ref := fmt.Sprintf("@e%d", i+1)
		h.refMap[ref] = el.Selector
		lines = append(lines, fmt.Sprintf("%s %s", ref, el.Label))
	}

	text := strings.Join(lines, "\n")
	if text == "" {
		text = "No elements found"
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserWait waits for an element to reach a specified state.
func (h *Handlers) browserWait(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	state := "attached"
	if s, ok := args["state"].(string); ok && s != "" {
		state = s
	}

	timeout := features.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	opts := features.WaitOptions{Timeout: timeout}

	switch state {
	case "attached":
		if err := features.WaitForSelector(h.client, "", selector, opts); err != nil {
			return nil, err
		}
	case "visible":
		if err := features.WaitForSelector(h.client, "", selector, opts); err != nil {
			return nil, err
		}
		visible, err := features.CheckVisible(h.client, "", selector)
		if err != nil {
			return nil, fmt.Errorf("visibility check failed: %w", err)
		}
		if !visible {
			return nil, fmt.Errorf("element %q found but not visible", selector)
		}
	case "hidden":
		if err := features.WaitForHidden(h.client, "", selector, opts); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid state: %q (use \"attached\", \"visible\", or \"hidden\")", state)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Element %q reached state: %s", selector, state),
		}},
	}, nil
}

// browserGetText returns the text content of the page or an element.
func (h *Handlers) browserGetText(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	var expr string
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		expr = fmt.Sprintf(`document.querySelector(%q)?.innerText || ''`, selector)
	} else {
		expr = `document.body.innerText`
	}

	result, err := h.client.Evaluate("", expr)
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}

	text := ""
	if result != nil {
		text = fmt.Sprintf("%v", result)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserGetURL returns the current page URL.
func (h *Handlers) browserGetURL(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	result, err := h.client.Evaluate("", "window.location.href")
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	url := ""
	if result != nil {
		url = fmt.Sprintf("%v", result)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: url,
		}},
	}, nil
}

// browserGetTitle returns the current page title.
func (h *Handlers) browserGetTitle(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	result, err := h.client.Evaluate("", "document.title")
	if err != nil {
		return nil, fmt.Errorf("failed to get title: %w", err)
	}

	title := ""
	if result != nil {
		title = fmt.Sprintf("%v", result)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: title,
		}},
	}, nil
}

// pageClockInstall installs a fake clock on the page.
func (h *Handlers) pageClockInstall(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	_, err := h.client.CallFunction("", clockInstallScript(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to install clock: %w", err)
	}

	if timeVal, ok := args["time"].(float64); ok {
		script := fmt.Sprintf("() => { window.__vibiumClock.setSystemTime(%v); return 'ok'; }", timeVal)
		if _, err := h.client.CallFunction("", script, nil); err != nil {
			return nil, fmt.Errorf("failed to set initial time: %w", err)
		}
	}

	if tz, ok := args["timezone"].(string); ok && tz != "" {
		if err := h.setTimezoneOverride(tz); err != nil {
			return nil, fmt.Errorf("failed to set timezone: %w", err)
		}
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: "Clock installed"}},
	}, nil
}

// pageClockFastForward fast-forwards the fake clock.
func (h *Handlers) pageClockFastForward(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	ticks, ok := args["ticks"].(float64)
	if !ok {
		return nil, fmt.Errorf("ticks is required")
	}

	script := fmt.Sprintf("() => { window.__vibiumClock.fastForward(%v); return 'ok'; }", ticks)
	if _, err := h.client.CallFunction("", script, nil); err != nil {
		return nil, fmt.Errorf("clock.fastForward failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Fast-forwarded %v ms", ticks)}},
	}, nil
}

// pageClockRunFor advances the fake clock, firing all callbacks.
func (h *Handlers) pageClockRunFor(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	ticks, ok := args["ticks"].(float64)
	if !ok {
		return nil, fmt.Errorf("ticks is required")
	}

	script := fmt.Sprintf("() => { window.__vibiumClock.runFor(%v); return 'ok'; }", ticks)
	if _, err := h.client.CallFunction("", script, nil); err != nil {
		return nil, fmt.Errorf("clock.runFor failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Ran for %v ms", ticks)}},
	}, nil
}

// pageClockPauseAt pauses the fake clock at a specific time.
func (h *Handlers) pageClockPauseAt(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeVal, ok := args["time"].(float64)
	if !ok {
		return nil, fmt.Errorf("time is required")
	}

	script := fmt.Sprintf("() => { window.__vibiumClock.pauseAt(%v); return 'ok'; }", timeVal)
	if _, err := h.client.CallFunction("", script, nil); err != nil {
		return nil, fmt.Errorf("clock.pauseAt failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Paused at %v", timeVal)}},
	}, nil
}

// pageClockResume resumes real-time progression.
func (h *Handlers) pageClockResume(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if _, err := h.client.CallFunction("", "() => { window.__vibiumClock.resume(); return 'ok'; }", nil); err != nil {
		return nil, fmt.Errorf("clock.resume failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: "Clock resumed"}},
	}, nil
}

// pageClockSetFixedTime freezes Date.now() at a value.
func (h *Handlers) pageClockSetFixedTime(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeVal, ok := args["time"].(float64)
	if !ok {
		return nil, fmt.Errorf("time is required")
	}

	script := fmt.Sprintf("() => { window.__vibiumClock.setFixedTime(%v); return 'ok'; }", timeVal)
	if _, err := h.client.CallFunction("", script, nil); err != nil {
		return nil, fmt.Errorf("clock.setFixedTime failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Fixed time set to %v", timeVal)}},
	}, nil
}

// pageClockSetSystemTime sets Date.now() without triggering timers.
func (h *Handlers) pageClockSetSystemTime(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeVal, ok := args["time"].(float64)
	if !ok {
		return nil, fmt.Errorf("time is required")
	}

	script := fmt.Sprintf("() => { window.__vibiumClock.setSystemTime(%v); return 'ok'; }", timeVal)
	if _, err := h.client.CallFunction("", script, nil); err != nil {
		return nil, fmt.Errorf("clock.setSystemTime failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("System time set to %v", timeVal)}},
	}, nil
}

// pageClockSetTimezone overrides or resets the browser timezone.
func (h *Handlers) pageClockSetTimezone(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	tz, _ := args["timezone"].(string)

	if tz == "" {
		// Reset to default
		if err := h.clearTimezoneOverride(); err != nil {
			return nil, fmt.Errorf("failed to clear timezone: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{Type: "text", Text: "Timezone reset to system default"}},
		}, nil
	}

	if err := h.setTimezoneOverride(tz); err != nil {
		return nil, fmt.Errorf("failed to set timezone: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Timezone set to %s", tz)}},
	}, nil
}

// setTimezoneOverride uses BiDi emulation.setTimezoneOverride.
func (h *Handlers) setTimezoneOverride(timezone string) error {
	tree, err := h.client.GetTree()
	if err != nil {
		return fmt.Errorf("failed to get browsing context: %w", err)
	}
	if len(tree.Contexts) == 0 {
		return fmt.Errorf("no browsing contexts available")
	}

	_, err = h.client.SendCommand("emulation.setTimezoneOverride", map[string]interface{}{
		"timezone": timezone,
		"contexts": []interface{}{tree.Contexts[0].Context},
	})
	return err
}

// clearTimezoneOverride resets the browser timezone to the system default.
func (h *Handlers) clearTimezoneOverride() error {
	tree, err := h.client.GetTree()
	if err != nil {
		return fmt.Errorf("failed to get browsing context: %w", err)
	}
	if len(tree.Contexts) == 0 {
		return fmt.Errorf("no browsing contexts available")
	}

	_, err = h.client.SendCommand("emulation.setTimezoneOverride", map[string]interface{}{
		"timezone": nil,
		"contexts": []interface{}{tree.Contexts[0].Context},
	})
	return err
}

// clockInstallScript returns the JS that installs the fake clock on the page.
// This is the same script used by the proxy handlers (defined separately to avoid circular imports).
func clockInstallScript() string {
	return `() => {
	if (window.__vibiumClock) return 'already_installed';

	const OrigDate = Date;
	const origSetTimeout = setTimeout;
	const origClearTimeout = clearTimeout;
	const origSetInterval = setInterval;
	const origClearInterval = clearInterval;
	const origRAF = requestAnimationFrame;
	const origCAF = cancelAnimationFrame;
	const origPerfNow = performance.now.bind(performance);

	let currentTime = OrigDate.now();
	let fixedTime = null;
	let paused = false;
	let nextId = 1;
	let resumeTimer = null;
	const timers = new Map();

	class FakeDate extends OrigDate {
		constructor(...args) {
			if (args.length === 0) {
				super(fixedTime !== null ? fixedTime : currentTime);
			} else {
				super(...args);
			}
		}
		static now() {
			return fixedTime !== null ? fixedTime : currentTime;
		}
		static parse(s) { return OrigDate.parse(s); }
		static UTC(...args) { return OrigDate.UTC(...args); }
	}

	function fakeSetTimeout(fn, delay, ...args) {
		if (typeof fn !== 'function') return 0;
		const id = nextId++;
		timers.set(id, {
			callback: fn, args: args,
			triggerTime: currentTime + (delay || 0),
			interval: 0, type: 'timeout'
		});
		return id;
	}

	function fakeSetInterval(fn, delay, ...args) {
		if (typeof fn !== 'function') return 0;
		const id = nextId++;
		timers.set(id, {
			callback: fn, args: args,
			triggerTime: currentTime + (delay || 0),
			interval: delay || 0, type: 'interval'
		});
		return id;
	}

	function fakeClearTimeout(id) { timers.delete(id); }
	function fakeClearInterval(id) { timers.delete(id); }

	let rafId = 1;
	const rafCallbacks = new Map();
	function fakeRAF(fn) { const id = rafId++; rafCallbacks.set(id, fn); return id; }
	function fakeCAF(id) { rafCallbacks.delete(id); }

	const startPerfTime = origPerfNow();
	const startCurrentTime = currentTime;
	function fakePerfNow() { return startPerfTime + (currentTime - startCurrentTime); }

	window.Date = FakeDate;
	window.setTimeout = fakeSetTimeout;
	window.setInterval = fakeSetInterval;
	window.clearTimeout = fakeClearTimeout;
	window.clearInterval = fakeClearInterval;
	window.requestAnimationFrame = fakeRAF;
	window.cancelAnimationFrame = fakeCAF;
	performance.now = fakePerfNow;

	function getDueTimers(upTo) {
		const due = [];
		for (const [id, t] of timers) {
			if (t.triggerTime <= upTo) due.push([id, t]);
		}
		due.sort((a, b) => a[1].triggerTime - b[1].triggerTime);
		return due;
	}

	function fireRAFs() {
		const cbs = Array.from(rafCallbacks.entries());
		rafCallbacks.clear();
		for (const [, fn] of cbs) { try { fn(currentTime); } catch (e) {} }
	}

	const clock = {
		fastForward(ms) {
			const target = currentTime + ms;
			currentTime = target;
			const due = getDueTimers(target);
			for (const [id, t] of due) {
				timers.delete(id);
				try { t.callback(...t.args); } catch (e) {}
			}
			fireRAFs();
		},
		runFor(ms) {
			const target = currentTime + ms;
			while (currentTime < target) {
				let earliest = null;
				let earliestId = null;
				for (const [id, t] of timers) {
					if (t.triggerTime <= target && (!earliest || t.triggerTime < earliest.triggerTime)) {
						earliest = t; earliestId = id;
					}
				}
				if (!earliest || earliest.triggerTime > target) { currentTime = target; break; }
				currentTime = earliest.triggerTime;
				if (earliest.type === 'interval' && earliest.interval > 0) {
					earliest.triggerTime = currentTime + earliest.interval;
				} else { timers.delete(earliestId); }
				try { earliest.callback(...earliest.args); } catch (e) {}
			}
			currentTime = target;
			fireRAFs();
		},
		pauseAt(time) {
			currentTime = time; paused = true;
			if (resumeTimer) { origClearInterval(resumeTimer); resumeTimer = null; }
			const due = getDueTimers(time);
			for (const [id, t] of due) { timers.delete(id); try { t.callback(...t.args); } catch (e) {} }
		},
		resume() {
			if (resumeTimer) return;
			paused = false;
			let lastReal = OrigDate.now();
			resumeTimer = origSetInterval(() => {
				const now = OrigDate.now();
				const delta = now - lastReal;
				lastReal = now;
				currentTime += delta;
				const due = getDueTimers(currentTime);
				for (const [id, t] of due) {
					if (t.type === 'interval' && t.interval > 0) { t.triggerTime = currentTime + t.interval; }
					else { timers.delete(id); }
					try { t.callback(...t.args); } catch (e) {}
				}
				fireRAFs();
			}, 16);
		},
		setFixedTime(time) { fixedTime = time; },
		setSystemTime(time) { currentTime = time; fixedTime = null; }
	};

	window.__vibiumClock = clock;
	return 'installed';
}`
}

// pollCallFunction polls a JS function until it returns a non-null/non-empty result.
func pollCallFunction(h *Handlers, script string, args []interface{}, timeout time.Duration) (interface{}, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		result, err := h.client.CallFunction("", script, args)
		if err == nil && result != nil {
			s := fmt.Sprintf("%v", result)
			if s != "" && s != "null" && s != "<nil>" {
				return result, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %s", timeout)
		}

		time.Sleep(interval)
	}
}

// browserFill clears an input field and types new text.
func (h *Handlers) browserFill(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required")
	}

	// Wait for element to be editable
	opts := features.DefaultWaitOptions()
	if err := features.WaitForType(h.client, "", selector, opts); err != nil {
		return nil, err
	}

	// Clear the field using JS
	clearScript := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		el.focus();
		el.value = '';
		el.dispatchEvent(new Event('input', {bubbles: true}));
		return 'cleared';
	}`
	result, err := h.client.CallFunction("", clearScript, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to clear field: %w", err)
	}
	if fmt.Sprintf("%v", result) == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// Click to ensure focus
	if err := h.client.ClickElement("", selector); err != nil {
		return nil, fmt.Errorf("failed to focus element: %w", err)
	}

	// Type the new text
	if err := h.client.TypeIntoElement("", selector, text); err != nil {
		return nil, fmt.Errorf("failed to type: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Filled %q into %s", text, selector),
		}},
	}, nil
}

// browserPress presses a key on a specific element or the focused element.
func (h *Handlers) browserPress(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// If selector given, click to focus first
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		opts := features.DefaultWaitOptions()
		if err := features.WaitForClick(h.client, "", selector, opts); err != nil {
			return nil, err
		}
		if err := h.client.ClickElement("", selector); err != nil {
			return nil, fmt.Errorf("failed to click element: %w", err)
		}
	}

	if err := h.client.PressKeyCombo("", key); err != nil {
		return nil, fmt.Errorf("failed to press key: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Pressed %s", key),
		}},
	}, nil
}

// browserBack navigates back in history.
func (h *Handlers) browserBack(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if err := h.client.TraverseHistory("", -1); err != nil {
		return nil, fmt.Errorf("failed to go back: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Navigated back",
		}},
	}, nil
}

// browserForward navigates forward in history.
func (h *Handlers) browserForward(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if err := h.client.TraverseHistory("", 1); err != nil {
		return nil, fmt.Errorf("failed to go forward: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Navigated forward",
		}},
	}, nil
}

// browserReload reloads the current page.
func (h *Handlers) browserReload(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if err := h.client.Reload(""); err != nil {
		return nil, fmt.Errorf("failed to reload: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Page reloaded",
		}},
	}, nil
}

// browserGetValue gets the current value of a form element.
func (h *Handlers) browserGetValue(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	value, err := h.client.GetElementValue("", selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: value,
		}},
	}, nil
}

// browserGetAttribute gets an HTML attribute value from an element.
func (h *Handlers) browserGetAttribute(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	attribute, ok := args["attribute"].(string)
	if !ok || attribute == "" {
		return nil, fmt.Errorf("attribute is required")
	}

	expr := fmt.Sprintf(`document.querySelector(%q)?.getAttribute(%q)`, selector, attribute)
	result, err := h.client.Evaluate("", expr)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}

	text := "null"
	if result != nil {
		text = fmt.Sprintf("%v", result)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserIsVisible checks if an element is visible on the page.
func (h *Handlers) browserIsVisible(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	visible, err := features.CheckVisible(h.client, "", selector)
	if err != nil {
		// Element not found or error — return false, not an error
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "false",
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", visible),
		}},
	}, nil
}

// browserCheck checks a checkbox or radio button (idempotent).
func (h *Handlers) browserCheck(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	// Check current state
	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		return el.checked ? 'checked' : 'unchecked';
	}`
	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to check state: %w", err)
	}

	state := fmt.Sprintf("%v", result)
	if state == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if state == "unchecked" {
		opts := features.DefaultWaitOptions()
		if err := features.WaitForClick(h.client, "", selector, opts); err != nil {
			return nil, err
		}
		if err := h.client.ClickElement("", selector); err != nil {
			return nil, fmt.Errorf("failed to click checkbox: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Checked %s", selector),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Already checked: %s", selector),
		}},
	}, nil
}

// browserUncheck unchecks a checkbox (idempotent).
func (h *Handlers) browserUncheck(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	// Check current state
	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		return el.checked ? 'checked' : 'unchecked';
	}`
	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to check state: %w", err)
	}

	state := fmt.Sprintf("%v", result)
	if state == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if state == "checked" {
		opts := features.DefaultWaitOptions()
		if err := features.WaitForClick(h.client, "", selector, opts); err != nil {
			return nil, err
		}
		if err := h.client.ClickElement("", selector); err != nil {
			return nil, fmt.Errorf("failed to click checkbox: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Unchecked %s", selector),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Already unchecked: %s", selector),
		}},
	}, nil
}

// browserScrollIntoView scrolls an element into view.
func (h *Handlers) browserScrollIntoView(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		el.scrollIntoView({behavior: 'instant', block: 'center'});
		return 'ok';
	}`
	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to scroll into view: %w", err)
	}

	if fmt.Sprintf("%v", result) == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Scrolled %s into view", selector),
		}},
	}, nil
}

// browserWaitForURL waits until the page URL contains a pattern.
func (h *Handlers) browserWaitForURL(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	timeout := features.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		result, err := h.client.Evaluate("", "window.location.href")
		if err == nil && result != nil {
			url := fmt.Sprintf("%v", result)
			if strings.Contains(url, pattern) {
				return &ToolsCallResult{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("URL matches pattern %q: %s", pattern, url),
					}},
				}, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for URL to contain %q", pattern)
		}

		time.Sleep(interval)
	}
}

// browserWaitForLoad waits until document.readyState is "complete".
func (h *Handlers) browserWaitForLoad(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeout := features.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		result, err := h.client.Evaluate("", "document.readyState")
		if err == nil && result != nil {
			state := fmt.Sprintf("%v", result)
			if state == "complete" {
				return &ToolsCallResult{
					Content: []Content{{
						Type: "text",
						Text: "Page loaded (readyState: complete)",
					}},
				}, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for page to load")
		}

		time.Sleep(interval)
	}
}

// browserSleep pauses execution for a specified number of milliseconds.
func (h *Handlers) browserSleep(args map[string]interface{}) (*ToolsCallResult, error) {
	ms, ok := args["ms"].(float64)
	if !ok || ms <= 0 {
		return nil, fmt.Errorf("ms is required and must be positive")
	}

	// Cap at 30 seconds
	if ms > 30000 {
		ms = 30000
	}

	time.Sleep(time.Duration(ms) * time.Millisecond)

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Slept for %v ms", ms),
		}},
	}, nil
}

// ensureBrowser checks that a browser session is active.
// If no browser is running, it auto-launches one (lazy launch).
func (h *Handlers) ensureBrowser() error {
	if h.client == nil {
		_, err := h.browserLaunch(map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("auto-launch failed: %w", err)
		}
	}
	return nil
}

// resolveSelector resolves @ref selectors to CSS selectors from the refMap.
func (h *Handlers) resolveSelector(selector string) string {
	if strings.HasPrefix(selector, "@e") {
		if resolved, ok := h.refMap[selector]; ok {
			return resolved
		}
	}
	return selector
}

// GetSelectorJS returns the JS getSelector(el) function body that generates unique CSS selectors.
func GetSelectorJS() string {
	return `function getSelector(el) {
			if (el.id) return '#' + CSS.escape(el.id);
			const parts = [];
			let cur = el;
			while (cur && cur !== document.body && cur !== document.documentElement) {
				let seg = cur.tagName.toLowerCase();
				if (cur.id) {
					parts.unshift('#' + CSS.escape(cur.id));
					break;
				}
				const parent = cur.parentElement;
				if (parent) {
					const siblings = Array.from(parent.children).filter(c => c.tagName === cur.tagName);
					if (siblings.length > 1) {
						const idx = siblings.indexOf(cur) + 1;
						seg += ':nth-of-type(' + idx + ')';
					}
				}
				parts.unshift(seg);
				cur = parent;
			}
			if (parts.length === 0) return el.tagName.toLowerCase();
			if (!parts[0].startsWith('#')) parts.unshift('body');
			return parts.join(' > ');
		}`
}

// GetLabelJS returns the JS getLabel(el) function body that generates descriptive labels.
func GetLabelJS() string {
	return `function getLabel(el) {
			const tag = el.tagName.toLowerCase();
			const type = el.getAttribute('type');
			let desc = '[' + tag;
			if (type) desc += ' type="' + type + '"';
			desc += ']';

			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return desc + ' "' + ariaLabel.substring(0, 60) + '"';

			const placeholder = el.getAttribute('placeholder');
			if (placeholder) return desc + ' placeholder="' + placeholder.substring(0, 60) + '"';

			const title = el.getAttribute('title');
			if (title) return desc + ' title="' + title.substring(0, 60) + '"';

			const text = (el.textContent || '').trim().substring(0, 60);
			if (text) return desc + ' "' + text + '"';

			const name = el.getAttribute('name');
			if (name) return desc + ' name="' + name + '"';

			const src = el.getAttribute('src');
			if (src) return desc + ' src="' + src.substring(0, 60) + '"';

			return desc;
		}`
}

// mapScript returns the JS function that maps interactive elements with refs.
// When a selector is provided, only elements within the matching subtree are returned.
func mapScript() string {
	return `(scopeSelector) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `

		const interactive = 'a[href], button, input, textarea, select, [role="button"], [role="link"], [role="checkbox"], [role="radio"], [role="tab"], [role="menuitem"], [role="switch"], [onclick], [tabindex]:not([tabindex="-1"]), summary, details';

		const root = scopeSelector ? document.querySelector(scopeSelector) : document;
		if (!root) return JSON.stringify([]);
		const els = root.querySelectorAll(interactive);
		const results = [];
		const seen = new Set();

		for (const el of els) {
			const style = window.getComputedStyle(el);
			if (style.display === 'none' || style.visibility === 'hidden' || el.offsetWidth === 0) continue;

			const sel = getSelector(el);
			if (seen.has(sel)) continue;
			seen.add(sel);

			results.push({ selector: sel, label: getLabel(el) });
		}

		return JSON.stringify(results);
	}`
}

// browserMap maps interactive elements with @refs.
func (h *Handlers) browserMap(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	var scopeSelector interface{}
	if sel, ok := args["selector"].(string); ok && sel != "" {
		scopeSelector = sel
	}
	result, err := h.client.CallFunction("", mapScript(), []interface{}{scopeSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to map elements: %w", err)
	}

	resultStr := fmt.Sprintf("%v", result)

	var elements []struct {
		Selector string `json:"selector"`
		Label    string `json:"label"`
	}
	if err := json.Unmarshal([]byte(resultStr), &elements); err != nil {
		return nil, fmt.Errorf("failed to parse map results: %w", err)
	}

	// Build ref map and output
	h.refMap = make(map[string]string)
	var lines []string
	for i, el := range elements {
		ref := fmt.Sprintf("@e%d", i+1)
		h.refMap[ref] = el.Selector
		lines = append(lines, fmt.Sprintf("%s %s", ref, el.Label))
	}

	output := strings.Join(lines, "\n")
	if output == "" {
		output = "No interactive elements found"
	}
	h.lastMap = output

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

// browserDiffMap compares current page state vs last map.
func (h *Handlers) browserDiffMap(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.lastMap == "" {
		return nil, fmt.Errorf("no previous map to diff against — run browser_map first")
	}

	// Get current map
	prevMap := h.lastMap
	_, err := h.browserMap(args)
	if err != nil {
		return nil, err
	}
	currentMap := h.lastMap

	// Simple line-based diff
	prevLines := strings.Split(prevMap, "\n")
	currLines := strings.Split(currentMap, "\n")

	prevSet := make(map[string]bool)
	for _, l := range prevLines {
		prevSet[l] = true
	}
	currSet := make(map[string]bool)
	for _, l := range currLines {
		currSet[l] = true
	}

	var diff []string
	for _, l := range prevLines {
		if !currSet[l] {
			diff = append(diff, "- "+l)
		}
	}
	for _, l := range currLines {
		if !prevSet[l] {
			diff = append(diff, "+ "+l)
		}
	}

	output := strings.Join(diff, "\n")
	if output == "" {
		output = "No changes detected"
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

// browserPDF saves the page as PDF.
func (h *Handlers) browserPDF(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	base64Data, err := h.client.PrintToPDF("")
	if err != nil {
		return nil, fmt.Errorf("failed to print PDF: %w", err)
	}

	// If filename provided, save to file
	if filename, ok := args["filename"].(string); ok && filename != "" {
		pdfData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode PDF: %w", err)
		}
		if err := os.WriteFile(filename, pdfData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save PDF: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("PDF saved to %s (%d bytes)", filename, len(pdfData)),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: base64Data,
		}},
	}, nil
}

// browserHighlight highlights an element with a visual overlay.
func (h *Handlers) browserHighlight(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		const prev = el.style.cssText;
		el.style.outline = '3px solid red';
		el.style.outlineOffset = '2px';
		el.style.backgroundColor = 'rgba(255,0,0,0.1)';
		setTimeout(() => { el.style.cssText = prev; }, 3000);
		return 'highlighted';
	}`

	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to highlight: %w", err)
	}

	if fmt.Sprintf("%v", result) == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Highlighted %s (3 seconds)", selector),
		}},
	}, nil
}

// browserDblClick double-clicks an element.
func (h *Handlers) browserDblClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	info, err := h.client.FindElement("", selector)
	if err != nil {
		return nil, err
	}

	x, y := info.GetCenter()
	if err := h.client.DoubleClick("", x, y); err != nil {
		return nil, fmt.Errorf("failed to double-click: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Double-clicked element: %s", selector),
		}},
	}, nil
}

// browserFocus focuses an element.
func (h *Handlers) browserFocus(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		el.focus();
		return 'focused';
	}`

	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to focus: %w", err)
	}

	if fmt.Sprintf("%v", result) == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Focused element: %s", selector),
		}},
	}, nil
}

// browserCount counts matching elements.
func (h *Handlers) browserCount(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	expr := fmt.Sprintf(`document.querySelectorAll(%q).length`, selector)
	result, err := h.client.Evaluate("", expr)
	if err != nil {
		return nil, fmt.Errorf("failed to count: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", result),
		}},
	}, nil
}

// browserIsEnabled checks if an element is enabled.
func (h *Handlers) browserIsEnabled(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		return el.disabled === true ? 'false' : 'true';
	}`

	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to check enabled: %w", err)
	}

	resultStr := fmt.Sprintf("%v", result)
	if resultStr == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: resultStr,
		}},
	}, nil
}

// browserIsChecked checks if an element is checked.
func (h *Handlers) browserIsChecked(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		return el.checked ? 'true' : 'false';
	}`

	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to check checked state: %w", err)
	}

	resultStr := fmt.Sprintf("%v", result)
	if resultStr == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: resultStr,
		}},
	}, nil
}

// browserWaitForText waits until text appears on the page.
func (h *Handlers) browserWaitForText(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	text, ok := args["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}

	timeout := features.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		result, err := h.client.Evaluate("", "document.body.innerText")
		if err == nil && result != nil {
			pageText := fmt.Sprintf("%v", result)
			if strings.Contains(pageText, text) {
				return &ToolsCallResult{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("Text %q found on page", text),
					}},
				}, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for text %q to appear", text)
		}

		time.Sleep(interval)
	}
}

// browserWaitForFn waits until a JS expression returns truthy.
func (h *Handlers) browserWaitForFn(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	timeout := features.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		result, err := h.client.Evaluate("", expression)
		if err == nil && result != nil {
			s := fmt.Sprintf("%v", result)
			if s != "" && s != "false" && s != "null" && s != "undefined" && s != "0" && s != "<nil>" {
				return &ToolsCallResult{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("Expression returned truthy: %s", s),
					}},
				}, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for expression to return truthy: %s", expression)
		}

		time.Sleep(interval)
	}
}

// browserDialogAccept accepts a dialog (alert, confirm, prompt).
func (h *Handlers) browserDialogAccept(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	text, _ := args["text"].(string)

	if err := h.client.HandleUserPrompt("", true, text); err != nil {
		return nil, fmt.Errorf("failed to accept dialog: %w", err)
	}

	msg := "Dialog accepted"
	if text != "" {
		msg = fmt.Sprintf("Dialog accepted with text: %q", text)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserDialogDismiss dismisses a dialog.
func (h *Handlers) browserDialogDismiss(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if err := h.client.HandleUserPrompt("", false, ""); err != nil {
		return nil, fmt.Errorf("failed to dismiss dialog: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Dialog dismissed",
		}},
	}, nil
}

// browserGetCookies returns all cookies.
func (h *Handlers) browserGetCookies(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	cookies, err := h.client.GetCookies("")
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	if len(cookies) == 0 {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No cookies",
			}},
		}, nil
	}

	var lines []string
	for _, c := range cookies {
		lines = append(lines, fmt.Sprintf("%s=%s (domain=%s, path=%s)", c.Name, c.Value, c.Domain, c.Path))
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: strings.Join(lines, "\n"),
		}},
	}, nil
}

// browserSetCookie sets a cookie.
func (h *Handlers) browserSetCookie(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	value, ok := args["value"].(string)
	if !ok {
		return nil, fmt.Errorf("value is required")
	}

	cookie := bidi.Cookie{
		Name:  name,
		Value: value,
	}
	if domain, ok := args["domain"].(string); ok {
		cookie.Domain = domain
	}
	if path, ok := args["path"].(string); ok {
		cookie.Path = path
	}

	if err := h.client.SetCookie("", cookie); err != nil {
		return nil, fmt.Errorf("failed to set cookie: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Cookie set: %s=%s", name, value),
		}},
	}, nil
}

// browserDeleteCookies deletes cookies.
func (h *Handlers) browserDeleteCookies(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	name, _ := args["name"].(string)

	if err := h.client.DeleteCookies("", name); err != nil {
		return nil, fmt.Errorf("failed to delete cookies: %w", err)
	}

	msg := "All cookies deleted"
	if name != "" {
		msg = fmt.Sprintf("Cookie %q deleted", name)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserMouseMove moves the mouse to coordinates.
func (h *Handlers) browserMouseMove(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	x, ok := args["x"].(float64)
	if !ok {
		return nil, fmt.Errorf("x is required")
	}
	y, ok := args["y"].(float64)
	if !ok {
		return nil, fmt.Errorf("y is required")
	}

	if err := h.client.MoveMouse("", x, y); err != nil {
		return nil, fmt.Errorf("failed to move mouse: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Mouse moved to (%d, %d)", int(x), int(y)),
		}},
	}, nil
}

// browserMouseDown presses a mouse button.
func (h *Handlers) browserMouseDown(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	button := 0
	if b, ok := args["button"].(float64); ok {
		button = int(b)
	}

	if err := h.client.MouseDown("", button); err != nil {
		return nil, fmt.Errorf("failed to press mouse button: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Mouse button %d pressed", button),
		}},
	}, nil
}

// browserMouseUp releases a mouse button.
func (h *Handlers) browserMouseUp(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	button := 0
	if b, ok := args["button"].(float64); ok {
		button = int(b)
	}

	if err := h.client.MouseUp("", button); err != nil {
		return nil, fmt.Errorf("failed to release mouse button: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Mouse button %d released", button),
		}},
	}, nil
}

// browserMouseClick clicks at coordinates or at the current position.
func (h *Handlers) browserMouseClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	button := 0
	if b, ok := args["button"].(float64); ok {
		button = int(b)
	}

	var pointerActions []map[string]interface{}

	// If coordinates provided, move there first
	x, hasX := args["x"].(float64)
	y, hasY := args["y"].(float64)
	if hasX && hasY {
		pointerActions = append(pointerActions, map[string]interface{}{
			"type":     "pointerMove",
			"x":        int(x),
			"y":        int(y),
			"duration": 0,
		})
	}

	pointerActions = append(pointerActions,
		map[string]interface{}{
			"type":   "pointerDown",
			"button": button,
		},
		map[string]interface{}{
			"type":   "pointerUp",
			"button": button,
		},
	)

	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": pointerActions,
		},
	}

	if err := h.client.PerformActions("", actions); err != nil {
		return nil, fmt.Errorf("failed to click: %w", err)
	}

	msg := "Clicked at current position"
	if hasX && hasY {
		msg = fmt.Sprintf("Clicked at (%d, %d)", int(x), int(y))
	}
	if button != 0 {
		msg += fmt.Sprintf(" with button %d", button)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserDrag drags from one element to another.
func (h *Handlers) browserDrag(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	source, ok := args["source"].(string)
	if !ok || source == "" {
		return nil, fmt.Errorf("source selector is required")
	}
	source = h.resolveSelector(source)

	target, ok := args["target"].(string)
	if !ok || target == "" {
		return nil, fmt.Errorf("target selector is required")
	}
	target = h.resolveSelector(target)

	if err := h.client.DragElement("", source, target); err != nil {
		return nil, fmt.Errorf("failed to drag: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Dragged %q to %q", source, target),
		}},
	}, nil
}

// browserSetViewport sets the viewport size.
func (h *Handlers) browserSetViewport(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	width, ok := args["width"].(float64)
	if !ok {
		return nil, fmt.Errorf("width is required")
	}
	height, ok := args["height"].(float64)
	if !ok {
		return nil, fmt.Errorf("height is required")
	}

	dpr := 0.0
	if d, ok := args["devicePixelRatio"].(float64); ok {
		dpr = d
	}

	if err := h.client.SetViewport("", int(width), int(height), dpr); err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	msg := fmt.Sprintf("Viewport set to %dx%d", int(width), int(height))
	if dpr > 0 {
		msg += fmt.Sprintf(" (DPR: %.1f)", dpr)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserGetViewport returns the current viewport dimensions.
func (h *Handlers) browserGetViewport(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	result, err := h.client.Evaluate("", "JSON.stringify({width: window.innerWidth, height: window.innerHeight, devicePixelRatio: window.devicePixelRatio})")
	if err != nil {
		return nil, fmt.Errorf("failed to get viewport: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", result),
		}},
	}, nil
}

// browserGetWindow returns the OS browser window state and dimensions.
// Uses BiDi browser.getClientWindows.
func (h *Handlers) browserGetWindow(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	resp, err := h.client.SendCommand("browser.getClientWindows", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get window: %w", err)
	}

	var getResult struct {
		ClientWindows []struct {
			State  string `json:"state"`
			X      int    `json:"x"`
			Y      int    `json:"y"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"clientWindows"`
	}
	if err := json.Unmarshal(resp.Result, &getResult); err != nil {
		return nil, fmt.Errorf("failed to parse window info: %w", err)
	}
	if len(getResult.ClientWindows) == 0 {
		return nil, fmt.Errorf("no client windows available")
	}

	win := getResult.ClientWindows[0]
	result := map[string]interface{}{
		"state":  win.State,
		"x":      win.X,
		"y":      win.Y,
		"width":  win.Width,
		"height": win.Height,
	}

	jsonBytes, _ := json.Marshal(result)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(jsonBytes),
		}},
	}, nil
}

// browserSetWindow sets the OS browser window size, position, or state.
// Uses chromedriver's classic WebDriver HTTP API.
func (h *Handlers) browserSetWindow(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	state, hasState := args["state"].(string)
	width, hasWidth := args["width"].(float64)
	height, hasHeight := args["height"].(float64)
	x, hasX := args["x"].(float64)
	y, hasY := args["y"].(float64)

	baseURL := fmt.Sprintf("http://localhost:%d/session/%s/window", h.launchResult.Port, h.launchResult.SessionID)

	// Handle named states (maximize, minimize, fullscreen) via dedicated endpoints
	if hasState && state != "normal" {
		endpoint := ""
		switch state {
		case "maximized":
			endpoint = baseURL + "/maximize"
		case "minimized":
			endpoint = baseURL + "/minimize"
		case "fullscreen":
			endpoint = baseURL + "/fullscreen"
		default:
			return nil, fmt.Errorf("unsupported window state: %s", state)
		}

		if err := h.chromedriverPost(endpoint, map[string]interface{}{}); err != nil {
			return nil, err
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Window state set to %s", state),
			}},
		}, nil
	}

	// For "normal" state or dimension changes, use /window/rect
	rect := map[string]interface{}{}
	if hasWidth {
		rect["width"] = int(width)
	}
	if hasHeight {
		rect["height"] = int(height)
	}
	if hasX {
		rect["x"] = int(x)
	}
	if hasY {
		rect["y"] = int(y)
	}

	if err := h.chromedriverPost(baseURL+"/rect", rect); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Window set to %dx%d", int(width), int(height))
	if hasX && hasY {
		msg += fmt.Sprintf(" at (%d, %d)", int(x), int(y))
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// chromedriverPost sends a POST request to a chromedriver classic WebDriver endpoint.
func (h *Handlers) chromedriverPost(url string, body map[string]interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("chromedriver request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chromedriver error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// browserEmulateMedia overrides CSS media features.
func (h *Handlers) browserEmulateMedia(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	overrides := make(map[string]string)
	if v, ok := args["media"].(string); ok && v != "" {
		overrides["media"] = v
	}
	if v, ok := args["colorScheme"].(string); ok && v != "" {
		overrides["colorScheme"] = v
	}
	if v, ok := args["reducedMotion"].(string); ok && v != "" {
		overrides["reducedMotion"] = v
	}
	if v, ok := args["forcedColors"].(string); ok && v != "" {
		overrides["forcedColors"] = v
	}
	if v, ok := args["contrast"].(string); ok && v != "" {
		overrides["contrast"] = v
	}

	if len(overrides) == 0 {
		return nil, fmt.Errorf("at least one media feature override is required")
	}

	overridesJSON, _ := json.Marshal(overrides)
	script := fmt.Sprintf(`(function() {
		const overrides = %s;
		if (!window.__vibiumMediaOverrides) window.__vibiumMediaOverrides = {};
		Object.assign(window.__vibiumMediaOverrides, overrides);
		const ov = window.__vibiumMediaOverrides;
		if (!window.__vibiumOrigMatchMedia) {
			window.__vibiumOrigMatchMedia = window.matchMedia.bind(window);
			window.matchMedia = function(query) {
				const orig = window.__vibiumOrigMatchMedia(query);
				const featureMap = {
					'prefers-color-scheme': ov.colorScheme,
					'prefers-reduced-motion': ov.reducedMotion,
					'forced-colors': ov.forcedColors,
					'prefers-contrast': ov.contrast
				};
				for (const [feature, value] of Object.entries(featureMap)) {
					if (!value) continue;
					const re = new RegExp('\\(' + feature + '\\s*:\\s*([^)]+)\\)');
					const m = query.match(re);
					if (m) {
						const requested = m[1].trim();
						const matches = requested === value;
						return {
							matches: matches,
							media: query,
							onchange: null,
							addEventListener: orig.addEventListener?.bind(orig) || function(){},
							removeEventListener: orig.removeEventListener?.bind(orig) || function(){},
							addListener: orig.addListener?.bind(orig) || function(){},
							removeListener: orig.removeListener?.bind(orig) || function(){}
						};
					}
				}
				return orig;
			};
		}
		return JSON.stringify({applied: Object.keys(overrides)});
	})()`, string(overridesJSON))

	result, err := h.client.Evaluate("", script)
	if err != nil {
		return nil, fmt.Errorf("failed to emulate media: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Media emulation applied: %v", result),
		}},
	}, nil
}

// browserSetGeolocation overrides the browser geolocation.
func (h *Handlers) browserSetGeolocation(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	latitude, ok := args["latitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("latitude is required")
	}
	longitude, ok := args["longitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("longitude is required")
	}

	accuracy := 1.0
	if a, ok := args["accuracy"].(float64); ok {
		accuracy = a
	}

	script := fmt.Sprintf(`(function() {
		const coords = {latitude: %f, longitude: %f, accuracy: %f};
		const position = {coords: coords, timestamp: Date.now()};
		navigator.geolocation.getCurrentPosition = function(success) { success(position); };
		navigator.geolocation.watchPosition = function(success) { success(position); return 0; };
		return JSON.stringify({set: true, latitude: coords.latitude, longitude: coords.longitude});
	})()`, latitude, longitude, accuracy)

	result, err := h.client.Evaluate("", script)
	if err != nil {
		return nil, fmt.Errorf("failed to set geolocation: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Geolocation set: %v", result),
		}},
	}, nil
}

// browserSetContent replaces the page HTML content.
func (h *Handlers) browserSetContent(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	html, ok := args["html"].(string)
	if !ok || html == "" {
		return nil, fmt.Errorf("html is required")
	}

	script := `(html) => { document.open(); document.write(html); document.close(); return 'ok'; }`
	_, err := h.client.CallFunction("", script, []interface{}{html})
	if err != nil {
		return nil, fmt.Errorf("failed to set content: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Page content set (%d chars)", len(html)),
		}},
	}, nil
}

// browserFrames lists all child frames (iframes) on the page.
func (h *Handlers) browserFrames(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	tree, err := h.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	if len(tree.Contexts) == 0 {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No browsing contexts",
			}},
		}, nil
	}

	// Collect all child frames from the first top-level context
	type frameInfo struct {
		Context string `json:"context"`
		URL     string `json:"url"`
		Name    string `json:"name,omitempty"`
	}

	var frames []frameInfo
	var collectFrames func(children []bidi.BrowsingContextInfo)
	collectFrames = func(children []bidi.BrowsingContextInfo) {
		for _, child := range children {
			fi := frameInfo{Context: child.Context, URL: child.URL}
			// Try to get frame name
			name, err := h.client.Evaluate(child.Context, "window.name")
			if err == nil && name != nil {
				fi.Name = fmt.Sprintf("%v", name)
			}
			frames = append(frames, fi)
			collectFrames(child.Children)
		}
	}
	collectFrames(tree.Contexts[0].Children)

	if len(frames) == 0 {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No frames found",
			}},
		}, nil
	}

	framesJSON, _ := json.Marshal(frames)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(framesJSON),
		}},
	}, nil
}

// browserFrame finds a frame by name or URL substring.
func (h *Handlers) browserFrame(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	nameOrURL, ok := args["nameOrUrl"].(string)
	if !ok || nameOrURL == "" {
		return nil, fmt.Errorf("nameOrUrl is required")
	}

	tree, err := h.client.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	if len(tree.Contexts) == 0 {
		return nil, fmt.Errorf("no browsing contexts")
	}

	type frameInfo struct {
		Context string `json:"context"`
		URL     string `json:"url"`
		Name    string `json:"name,omitempty"`
	}

	// Collect all child frames
	var frames []frameInfo
	var collectFrames func(children []bidi.BrowsingContextInfo)
	collectFrames = func(children []bidi.BrowsingContextInfo) {
		for _, child := range children {
			fi := frameInfo{Context: child.Context, URL: child.URL}
			name, err := h.client.Evaluate(child.Context, "window.name")
			if err == nil && name != nil {
				fi.Name = fmt.Sprintf("%v", name)
			}
			frames = append(frames, fi)
			collectFrames(child.Children)
		}
	}
	collectFrames(tree.Contexts[0].Children)

	// Try exact name match first
	for _, f := range frames {
		if f.Name == nameOrURL {
			result, _ := json.Marshal(f)
			return &ToolsCallResult{
				Content: []Content{{
					Type: "text",
					Text: string(result),
				}},
			}, nil
		}
	}

	// Try URL substring match
	for _, f := range frames {
		if strings.Contains(f.URL, nameOrURL) {
			result, _ := json.Marshal(f)
			return &ToolsCallResult{
				Content: []Content{{
					Type: "text",
					Text: string(result),
				}},
			}, nil
		}
	}

	return nil, fmt.Errorf("no frame matching %q", nameOrURL)
}

// browserUpload sets files on an input[type=file] element.
func (h *Handlers) browserUpload(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	filesRaw, ok := args["files"]
	if !ok {
		return nil, fmt.Errorf("files is required")
	}

	var files []string
	switch v := filesRaw.(type) {
	case []interface{}:
		for _, f := range v {
			if s, ok := f.(string); ok {
				files = append(files, s)
			}
		}
	default:
		return nil, fmt.Errorf("files must be an array of strings")
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file path is required")
	}

	if err := h.client.SetFiles("", selector, files); err != nil {
		return nil, fmt.Errorf("failed to set files: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Set %d file(s) on %s", len(files), selector),
		}},
	}, nil
}

// browserTraceStart starts trace recording.
func (h *Handlers) browserTraceStart(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if h.traceRecorder != nil {
		return nil, fmt.Errorf("trace already recording — stop it first")
	}

	name, _ := args["name"].(string)
	if name == "" {
		name = "trace"
	}
	screenshots, _ := args["screenshots"].(bool)
	snapshots, _ := args["snapshots"].(bool)

	h.traceRecorder = &traceRecorder{
		name:        name,
		screenshots: screenshots,
		snapshots:   snapshots,
		startTime:   time.Now(),
	}

	// Start screenshot capture loop in background
	if screenshots {
		h.traceRecorder.done = make(chan struct{})
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-h.traceRecorder.done:
					return
				case <-ticker.C:
					data, err := h.client.CaptureScreenshot("")
					if err == nil {
						h.traceRecorder.addScreenshot(data)
					}
				}
			}
		}()
	}

	// Capture snapshot if enabled
	if snapshots {
		html, err := h.client.Evaluate("", "document.documentElement.outerHTML")
		if err == nil && html != nil {
			h.traceRecorder.addSnapshot(fmt.Sprintf("%v", html))
		}
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Trace %q started (screenshots: %v, snapshots: %v)", name, screenshots, snapshots),
		}},
	}, nil
}

// browserTraceStop stops trace recording and saves to a ZIP file.
func (h *Handlers) browserTraceStop(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.traceRecorder == nil {
		return nil, fmt.Errorf("no trace recording in progress")
	}

	path, _ := args["path"].(string)
	if path == "" {
		path = "trace.zip"
	}

	// Capture final snapshot if enabled
	if h.traceRecorder.snapshots && h.client != nil {
		html, err := h.client.Evaluate("", "document.documentElement.outerHTML")
		if err == nil && html != nil {
			h.traceRecorder.addSnapshot(fmt.Sprintf("%v", html))
		}
	}

	// Stop screenshot loop
	if h.traceRecorder.done != nil {
		close(h.traceRecorder.done)
	}

	// Write ZIP
	if err := h.traceRecorder.writeZip(path); err != nil {
		h.traceRecorder = nil
		return nil, fmt.Errorf("failed to write trace: %w", err)
	}

	h.traceRecorder = nil

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Trace saved to %s", path),
		}},
	}, nil
}

// browserStorageState exports cookies, localStorage, and sessionStorage.
func (h *Handlers) browserStorageState(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	// Get cookies
	cookies, err := h.client.GetCookies("")
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	// Get localStorage and sessionStorage
	script := `JSON.stringify({
		origin: location.origin,
		localStorage: (function() {
			var ls = {};
			for (var i = 0; i < localStorage.length; i++) {
				var key = localStorage.key(i);
				ls[key] = localStorage.getItem(key);
			}
			return ls;
		})(),
		sessionStorage: (function() {
			var ss = {};
			for (var i = 0; i < sessionStorage.length; i++) {
				var key = sessionStorage.key(i);
				ss[key] = sessionStorage.getItem(key);
			}
			return ss;
		})()
	})`

	storageResult, err := h.client.Evaluate("", script)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	// Build combined state
	state := map[string]interface{}{
		"cookies": cookies,
		"storage": storageResult,
	}

	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(stateJSON),
		}},
	}, nil
}

// browserRestoreStorage restores cookies and storage from a JSON state.
func (h *Handlers) browserRestoreStorage(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state struct {
		Cookies []bidi.Cookie `json:"cookies"`
		Storage json.RawMessage `json:"storage"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Restore cookies
	for _, cookie := range state.Cookies {
		if err := h.client.SetCookie("", cookie); err != nil {
			log.Debug("failed to restore cookie", "name", cookie.Name, "error", err)
		}
	}

	// Restore localStorage/sessionStorage if present
	if len(state.Storage) > 0 {
		script := fmt.Sprintf(`(function() {
			var state = %s;
			if (state.localStorage) {
				for (var key in state.localStorage) {
					localStorage.setItem(key, state.localStorage[key]);
				}
			}
			if (state.sessionStorage) {
				for (var key in state.sessionStorage) {
					sessionStorage.setItem(key, state.sessionStorage[key]);
				}
			}
			return 'ok';
		})()`, string(state.Storage))
		h.client.Evaluate("", script)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Storage state restored from %s (%d cookies)", path, len(state.Cookies)),
		}},
	}, nil
}

// browserDownloadSetDir sets the download directory.
func (h *Handlers) browserDownloadSetDir(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	dir, ok := args["path"].(string)
	if !ok || dir == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Make absolute
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Get the connection to set download behavior
	params := map[string]interface{}{
		"downloadBehavior": map[string]interface{}{
			"type":              "allowed",
			"destinationFolder": absDir,
		},
	}

	if _, err := h.client.SendCommand("browser.setDownloadBehavior", params); err != nil {
		return nil, fmt.Errorf("failed to set download directory: %w", err)
	}

	h.downloadDir = absDir

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Download directory set to %s", absDir),
		}},
	}, nil
}
