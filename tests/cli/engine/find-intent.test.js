/**
 * CLI Tests: find by intent, interact by reference.
 * A bare string is a description, css is explicit, image matches locally,
 * and every match is an @eN ref the next command can use. No AI provider is
 * configured here, so the deterministic paths and their error messages are
 * what is asserted.
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require("../../helpers");
const { PATTERN_PNG, makePatternPNG } = require('../../helpers/test-server');

const env = { ...process.env, VIBIUM_LOAD_AI_ENV: '0' };
for (const key of Object.keys(env)) {
  if (key.startsWith('VIBIUM_AI_') || /_API_KEY$/.test(key)) delete env[key];
}

let serverProcess, baseURL, dir, pattern, absent;

function sh(cmd) {
  return execSync(`${VIBIUM} ${cmd}`, { encoding: 'utf-8', timeout: 60000, env });
}

function fails(cmd) {
  const r = spawnSync(VIBIUM, cmd, { encoding: 'utf-8', timeout: 60000, env });
  assert.notStrictEqual(r.status, 0, `expected failure: ${cmd.join(' ')}\n${r.stdout}`);
  return r.stderr + r.stdout;
}

function json(cmd) {
  return JSON.parse(sh(`${cmd} --json`));
}

const ref = (text) => text.match(/@e\d+/)[0];

before(async () => {
  // A separate process: execSync blocks this event loop, so an in-process
  // server could not answer the browser while a command runs.
  serverProcess = spawn('node', [path.join(__dirname, '../../helpers/test-server.js')], { stdio: ['pipe', 'pipe', 'pipe'] });
  baseURL = await new Promise((resolve) => serverProcess.stdout.once('data', (d) => resolve(d.toString().trim())));
  dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-find-'));
  pattern = path.join(dir, 'pattern.png');
  absent = path.join(dir, 'absent.png');
  fs.writeFileSync(pattern, PATTERN_PNG);
  fs.writeFileSync(absent, makePatternPNG(4242));
  sh(`go ${baseURL}/find-images`);
});

after(() => {
  if (serverProcess) serverProcess.kill();
  fs.rmSync(dir, { recursive: true, force: true });
});

describe('CLI: find by description', () => {
  test('a name in the page tree resolves without a model', () => {
    assert.match(sh(`find "Add to Cart"`), /@e\d+ \[button\] "Add to Cart"/);
    assert.match(sh(`find "the Learn more link"`), /@e\d+ \[a\] "Learn more"/);
    assert.match(sh(`find "the search box"`), /@e\d+ \[input type="search"\]/);
  });

  test('a bare string that reads like CSS is refused at once and names the explicit forms', () => {
    const started = Date.now();
    const out = fails(['find', 'h1']);
    assert.ok(Date.now() - started < 5000, 'must not wait out a timeout or call a model');
    assert.match(out, /reads like a CSS selector/);
    assert.match(out, /vibium find css "h1"/);
    assert.match(out, /find\(\{css: "h1"\}\)/);
    assert.match(out, /find\(css="h1"\)/);
    assert.match(out, /SelectorOptions\(\)\.css\("h1"\)/);
    assert.match(out, /vibium find description "h1"/);
    assert.match(fails(['find', 'description', 'h1', '--timeout', '500']), /none is configured/, 'an explicit description is searched for');
  });

  test('a description needing sight reports the missing model', () => {
    const out = fails(['find', 'the red button', '--timeout', '500']);
    assert.match(out, /needs a vision model/);
  });

  test('strategy vlm needs a provider', () => {
    assert.match(fails(['find', 'the search box', '--strategy', 'vlm']), /needs a configured vision model/);
  });

  test('strategy tree never consults a model', () => {
    assert.match(sh(`find "Add to Cart" --strategy tree`), /\[button\] "Add to Cart"/);
    assert.match(fails(['find', 'the red button', '--strategy', 'tree', '--timeout', '500']), /only matches names and roles/);
  });
});

describe('CLI: explicit locators', () => {
  test('css and tag are exact', () => {
    assert.match(sh(`find css h1`), /@e\d+ \[h1\] "Gallery"/);
    assert.match(sh(`find tag h1`), /@e\d+ \[h1\] "Gallery"/);
    assert.match(sh(`find css img --all`), /@e\d+ \[img\][\s\S]*@e\d+ \[img\]/);
  });

  test('an explicit locator rejects a model strategy', () => {
    assert.match(fails(['find', 'css', 'h1', '--strategy', 'cv']), /strategy cv needs an image/);
    assert.match(fails(['find', 'role', 'button', '--strategy', 'vlm']), /never by a model/);
  });

  test('refs are stable across finds on one page', () => {
    const first = ref(sh(`find css h1`));
    sh(`find css "p" --all`);
    assert.strictEqual(ref(sh(`find css h1`)), first, 'the same element keeps its ref');
    assert.strictEqual(ref(sh(`find "Gallery"`)), first, 'a description of it resolves to the same ref');
  });

  test('--json is an array of refs with boxes', () => {
    const out = json(`find css img --all`);
    assert.strictEqual(out.ok, true);
    assert.ok(Array.isArray(out.result) && out.result.length >= 3);
    for (const m of out.result) {
      assert.match(m.ref, /^@e\d+$/);
      assert.strictEqual(m.source, 'dom');
      assert.strictEqual(m.box.width, 96);
    }
  });
});

describe('CLI: find image', () => {
  test('finds every copy locally and snaps to the element under it', () => {
    const out = json(`find image ${pattern} --all`);
    assert.strictEqual(out.ok, true);
    assert.strictEqual(out.result.length, 3, JSON.stringify(out.result));
    for (const m of out.result) {
      assert.strictEqual(m.source, 'image');
      assert.strictEqual(m.tag, 'img');
      assert.strictEqual(m.box.width, 96);
      assert.strictEqual(m.box.height, 96);
    }
    assert.match(sh(`find image ${pattern}`), /@e\d+ \[img\] src="\/fixtures\/pattern.png"/);
  });

  test('--details reports the method and raw scores', () => {
    const m = json(`find image ${pattern} --details`).result[0];
    assert.strictEqual(m.details.method, 'sift');
    assert.ok(m.details.inliers >= 8, `inliers ${m.details.inliers}`);
    assert.ok(Math.abs(m.details.scale - 1) < 0.2, `scale ${m.details.scale}`);
  });

  test('no match without a provider is empty, not a configuration error', () => {
    const out = json(`find image ${absent} --all --timeout 500`);
    assert.deepStrictEqual(out, { ok: true, result: [] });
    const err = fails(['find', 'image', absent, '--timeout', '500']);
    assert.match(err, /no verified match/);
    assert.doesNotMatch(err, /set VIBIUM_AI_PROVIDER/);
  });

  test('strategy cv is local; vlm needs a provider', () => {
    assert.match(sh(`find image ${pattern} --strategy cv`), /@e\d+ \[img\]/);
    assert.match(fails(['find', 'image', pattern, '--strategy', 'vlm']), /needs a configured vision model/);
  });

  test('a missing file is an error', () => {
    assert.match(fails(['find', 'image', path.join(dir, 'nope.png')]), /cannot read image/);
  });

  test('a matched ref clicks the real element', () => {
    const r = ref(sh(`find image ${pattern}`));
    assert.match(sh(`click ${r}`), /Clicked element: #t1/);
  });

  test('tap and click by ref reach a button', () => {
    const r = ref(sh(`find "Add to Cart"`));
    sh(`tap ${r}`);
    sh(`click ${r}`);
    assert.strictEqual(sh(`eval window.__added`).trim(), '2');
  });
});

describe('CLI: visual refs', () => {
  let r;

  before(() => {
    sh(`go ${baseURL}/find-canvas`);
    sh(`wait fn "window.__drawn === true"`);
    const out = json(`find image ${pattern}`);
    r = out.result[0].ref;
    assert.strictEqual(out.result[0].selector, undefined, 'a region on a canvas has no element of its own');
    assert.strictEqual(out.result[0].source, 'image');
  });

  test('click lands on the box center', () => {
    sh(`click ${r}`);
    const clicks = JSON.parse(sh(`eval "JSON.stringify(window.__clicks)"`));
    assert.strictEqual(clicks.length, 1);
    assert.ok(Math.abs(clicks[0].x - 348) <= 2 && Math.abs(clicks[0].y - 198) <= 2, JSON.stringify(clicks));
  });

  test('a DOM-only command is refused', () => {
    assert.match(fails(['fill', r, 'x']), /visual match with no DOM element/);
  });

  test('a scroll re-detects the image before clicking', () => {
    sh(`eval "window.scrollTo(0, 40)"`);
    sh(`click ${r}`);
    assert.strictEqual(sh(`eval window.__clicks.length`).trim(), '2');
    sh(`highlight ${r}`);
    sh(`tap ${r}`);
    assert.strictEqual(sh(`eval window.__clicks.length`).trim(), '3');
  });

  test('a navigation forgets the page\'s refs', () => {
    sh(`go ${baseURL}/example`);
    assert.match(fails(['click', r]), /not a mapped element; run find or map first/);
  });
});
