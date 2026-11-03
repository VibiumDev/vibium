package codegen

import (
	"strings"
	"testing"
)

func action(method string, params map[string]interface{}) Step {
	return Step{Kind: StepAction, Title: method, Method: method, Params: params}
}

var sampleSteps = []Step{
	{Kind: StepGroupStart, Title: "login flow"},
	action("vibium:page.navigate", map[string]interface{}{"url": "https://example.com/login", "context": "abc"}),
	action("vibium:element.fill", map[string]interface{}{"selector": "#user", "value": "admin"}),
	action("vibium:element.press", map[string]interface{}{"selector": "#user", "key": "Tab"}),
	action("vibium:element.click", map[string]interface{}{"role": "button", "text": "Log in"}),
	{Kind: StepGroupEnd},
	action("vibium:element.text", map[string]interface{}{"selector": "#flash"}),
	Step{Kind: StepAction, Title: "Element.click", Method: "vibium:element.click",
		Params: map[string]interface{}{"selector": "#gone"}, Error: "no such element"},
	action("vibium:page.setViewport", map[string]interface{}{"width": float64(1280), "height": float64(720)}),
	action("vibium:keyboard.press", map[string]interface{}{"key": "Enter"}),
	action("vibium:mouse.click", map[string]interface{}{"x": float64(10), "y": 20.5}),
	action("vibium:element.click", map[string]interface{}{"selector": ".row", "index": float64(2)}),
	action("vibium:page.wait", map[string]interface{}{"ms": float64(500)}),
	action("vibium:page.frobnicate", map[string]interface{}{}),
}

const wantJS = `import { browser } from 'vibium';

const bro = await browser.start();
const vibe = await bro.page();

// group: login flow
await vibe.go('https://example.com/login');
const el1 = await vibe.find('#user');
await el1.fill('admin');
await el1.press('Tab');
await vibe.find({ role: 'button', text: 'Log in' }).click();
// skipped failed action: Element.click (no such element)
await vibe.setViewport({ width: 1280, height: 720 });
await vibe.keyboard.press('Enter');
await vibe.mouse.click(10, 20.5);
await (await vibe.findAll('.row'))[2].click();
await vibe.wait(500);
// skipped unsupported action: vibium:page.frobnicate

await bro.stop();
`

const wantPython = `from vibium import browser

bro = browser.start()
vibe = bro.page()

# group: login flow
vibe.go('https://example.com/login')
el1 = vibe.find('#user')
el1.fill('admin')
el1.press('Tab')
vibe.find(role='button', text='Log in').click()
# skipped failed action: Element.click (no such element)
vibe.set_viewport({'width': 1280, 'height': 720})
vibe.keyboard.press('Enter')
vibe.mouse.click(10, 20.5)
vibe.find_all('.row')[2].click()
vibe.wait(500)
# skipped unsupported action: vibium:page.frobnicate

bro.stop()
`

func TestGenerateJavaScript(t *testing.T) {
	got, err := Generate(sampleSteps, "javascript")
	if err != nil {
		t.Fatal(err)
	}
	if got != wantJS {
		t.Errorf("JavaScript output mismatch\ngot:\n%s\nwant:\n%s", got, wantJS)
	}
}

func TestGeneratePython(t *testing.T) {
	got, err := Generate(sampleSteps, "python")
	if err != nil {
		t.Fatal(err)
	}
	if got != wantPython {
		t.Errorf("Python output mismatch\ngot:\n%s\nwant:\n%s", got, wantPython)
	}
}

func TestGenerateRejectsUnknownLanguage(t *testing.T) {
	if _, err := Generate(nil, "ruby"); err == nil {
		t.Fatal("expected an error for an unsupported language")
	}
}

func TestGenerateSpecialActions(t *testing.T) {
	steps := []Step{
		action("vibium:element.set", map[string]interface{}{"selector": "#opt", "value": false}),
		action("vibium:element.set", map[string]interface{}{"selector": "#all", "value": true}),
		action("vibium:element.dragTo", map[string]interface{}{
			"selector": "#src",
			"target":   map[string]interface{}{"selector": "#dst"},
		}),
		action("vibium:element.setFiles", map[string]interface{}{"selector": "#up", "files": []interface{}{"a.txt", "b.txt"}}),
		action("vibium:element.dispatchEvent", map[string]interface{}{"selector": "#x", "eventType": "change"}),
		action("vibium:element.selectOption", map[string]interface{}{"selector": "#sel", "value": "two"}),
		action("vibium:element.waitFor", map[string]interface{}{"selector": "#late", "state": "visible"}),
		action("vibium:page.eval", map[string]interface{}{"expression": "1 + 1"}),
		action("vibium:touch.swipe", map[string]interface{}{"x1": float64(0), "y1": float64(0), "x2": float64(100), "y2": float64(200)}),
		action("vibium:element.highlight", map[string]interface{}{"selector": "#x"}),
		action("vibium:recording.start", nil),
	}

	js, err := Generate(steps, "javascript")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"await vibe.find('#opt').set(false);\n",
		"await vibe.find('#all').set();\n",
		"await vibe.find('#src').dragTo(await vibe.find('#dst'));\n",
		"await vibe.find('#up').setFiles(['a.txt', 'b.txt']);\n",
		"await vibe.find('#x').dispatchEvent('change');\n",
		"await vibe.find('#sel').selectOption('two');\n",
		"await vibe.find('#late').waitUntil('visible');\n",
		"await vibe.evaluate('1 + 1');\n",
		"await vibe.touch.swipe(0, 0, 100, 200);\n",
	} {
		if !contains(js, want) {
			t.Errorf("JavaScript output lacks %q\ngot:\n%s", want, js)
		}
	}
	for _, absent := range []string{"highlight", "recording"} {
		if contains(js, absent) {
			t.Errorf("JavaScript output should not contain %q\ngot:\n%s", absent, js)
		}
	}

	py, err := Generate(steps, "python")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"vibe.find('#opt').set(False)\n",
		"vibe.find('#all').set()\n",
		"vibe.find('#src').drag_to(vibe.find('#dst'))\n",
		"vibe.find('#up').set_files(['a.txt', 'b.txt'])\n",
		"vibe.find('#x').dispatch_event('change')\n",
		"vibe.find('#sel').select_option('two')\n",
		"vibe.find('#late').wait_until('visible')\n",
		"vibe.evaluate('1 + 1')\n",
		"vibe.touch.swipe(0, 0, 100, 200)\n",
	} {
		if !contains(py, want) {
			t.Errorf("Python output lacks %q\ngot:\n%s", want, py)
		}
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
