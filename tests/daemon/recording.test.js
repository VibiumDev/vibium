/**
 * Daemon CLI: Recording Tests
 * Tests record start/stop/group start/group stop/chunk start/chunk stop CLI commands.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

// --- CLI helpers (same pattern as cli-commands.test.js) ---

function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  const result = clicker(`--json ${args}`, opts);
  return JSON.parse(result);
}

// Like clickerJSON but handles non-zero exit codes (for testing error paths)
function clickerJSONSafe(args, opts = {}) {
  try {
    return clickerJSON(args, opts);
  } catch (e) {
    if (e.stdout) {
      return JSON.parse(e.stdout.trim());
    }
    throw e;
  }
}

function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

// --- Recording zip helpers ---

function unzipRecording(zipPath) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-daemon-rec-'));
  execSync(`unzip -o "${zipPath}" -d "${tmpDir}/extracted"`, { stdio: 'pipe' });
  return { tmpDir, extractedDir: path.join(tmpDir, 'extracted') };
}

function cleanupDir(dir) {
  fs.rmSync(dir, { recursive: true, force: true });
}

function readRecordingEvents(extractedDir) {
  const files = fs.readdirSync(extractedDir).filter(f => f.endsWith('.trace'));
  const events = [];
  for (const file of files) {
    const content = fs.readFileSync(path.join(extractedDir, file), 'utf-8');
    for (const line of content.split('\n')) {
      if (line.trim()) {
        events.push(JSON.parse(line));
      }
    }
  }
  return events;
}

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

describe('Daemon CLI: Recording', () => {
  const tmpFiles = [];
  const tmpDirs = [];

  function tmpPath(name) {
    const p = path.join(os.tmpdir(), `vibium-daemon-rec-${Date.now()}-${name}`);
    tmpFiles.push(p);
    return p;
  }

  before(() => {
    stopDaemon();
    clicker('daemon start --headless');
    clicker(`go ${baseURL}/example`);
  });

  after(() => {
    stopDaemon();
    for (const f of tmpFiles) {
      try { fs.unlinkSync(f); } catch {}
    }
    for (const d of tmpDirs) {
      try { cleanupDir(d); } catch {}
    }
  });

  test('record start begins recording', () => {
    const result = clickerJSON('record start');
    assert.strictEqual(result.ok, true, 'record start should succeed');
    assert.ok(result.result.includes('started'), 'Should confirm recording started');

    // Clean up
    const stopPath = tmpPath('start-test.zip');
    clickerJSON(`record stop -o "${stopPath}"`);
  });

  test('record start rejects when already recording', () => {
    clickerJSON('record start');

    const result = clickerJSONSafe('record start');
    assert.strictEqual(result.ok, false, 'Second start should fail');
    assert.ok(result.error.includes('already recording'), 'Should say already recording');

    // Clean up
    const stopPath = tmpPath('double-start.zip');
    clickerJSON(`record stop -o "${stopPath}"`);
  });

  test('record stop saves valid Playwright-compatible zip', () => {
    clickerJSON('record start');
    clickerJSON(`go ${baseURL}/example`);

    const zipPath = tmpPath('valid-zip.zip');
    const result = clickerJSON(`record stop -o "${zipPath}"`);
    assert.strictEqual(result.ok, true, 'record stop should succeed');
    assert.ok(result.result.includes('Recording saved'), 'Should confirm save');

    // Verify file
    assert.ok(fs.existsSync(zipPath), 'Zip file should exist');
    assert.ok(fs.statSync(zipPath).size > 0, 'Zip file should not be empty');

    // Unzip and verify
    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    tmpDirs.push(tmpDir);
    const files = fs.readdirSync(extractedDir);
    assert.ok(files.some(f => f.endsWith('.trace')), 'Should have .trace file');
    assert.ok(files.some(f => f.endsWith('.network')), 'Should have .network file');

    const events = readRecordingEvents(extractedDir);
    assert.ok(events.length > 0, 'Should have recording events');
    assert.strictEqual(events[0].type, 'context-options');
    assert.strictEqual(events[0].browserName, 'chromium');

    // Verify before/after action events (unified with API path)
    const beforeEvents = events.filter(e => e.type === 'before' && e.method !== 'group');
    assert.ok(beforeEvents.length > 0, 'Should have before events for actions');
    assert.ok(beforeEvents[0].callId, 'before event should have callId');
    assert.ok(beforeEvents[0].title, 'before event should have title');

    const afterEvents = events.filter(e => e.type === 'after');
    assert.ok(afterEvents.length > 0, 'Should have after events for actions');

    // Verify before/after callIds match
    const beforeCallId = beforeEvents[0].callId;
    const matchingAfter = afterEvents.find(e => e.callId === beforeCallId);
    assert.ok(matchingAfter, 'Should have matching after event for first before event');
  });

  test('record stop rejects when not recording', () => {
    const result = clickerJSONSafe('record stop');
    assert.strictEqual(result.ok, false, 'Stop without start should fail');
    assert.ok(result.error.includes('no recording'), 'Should say no recording in progress');
  });

  test('record start --screenshots captures per-action screenshots', () => {
    clickerJSON('record start --screenshots');
    clickerJSON(`go ${baseURL}/example`);
    clickerJSON('click "a"');

    const zipPath = tmpPath('screenshots.zip');
    clickerJSON(`record stop -o "${zipPath}"`);

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    tmpDirs.push(tmpDir);

    // Check resources directory
    const resourcesDir = path.join(extractedDir, 'resources');
    assert.ok(fs.existsSync(resourcesDir), 'resources/ directory should exist');
    const resources = fs.readdirSync(resourcesDir);
    assert.ok(resources.length > 0, 'Should have screenshot resources');

    // Check screencast-frame events
    const events = readRecordingEvents(extractedDir);
    const frames = events.filter(e => e.type === 'screencast-frame');
    assert.ok(frames.length > 0, 'Should have screencast-frame events');
    assert.ok(frames[0].sha1, 'screencast-frame should have sha1');
    assert.ok(frames[0].width > 0, 'screencast-frame should have width');
    assert.ok(frames[0].height > 0, 'screencast-frame should have height');
  });

  test('record group start / group stop adds group markers', () => {
    clickerJSON('record start');
    clickerJSON('record group start "Login"');
    clickerJSON(`go ${baseURL}/example`);
    clickerJSON('record group stop');

    const zipPath = tmpPath('groups.zip');
    clickerJSON(`record stop -o "${zipPath}"`);

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    tmpDirs.push(tmpDir);
    const events = readRecordingEvents(extractedDir);

    // Look for group before event
    const beforeEvents = events.filter(e => e.type === 'before' && e.title === 'Login' && e.method === 'tracingGroup');
    assert.ok(beforeEvents.length > 0, 'Should have before event with title "Login" and method "tracingGroup"');

    // Look for matching after event
    const callId = beforeEvents[0].callId;
    const afterEvents = events.filter(e => e.type === 'after' && e.callId === callId);
    assert.ok(afterEvents.length > 0, 'Should have matching after event with same callId');
  });

  test('record chunk start / chunk stop saves chunk zip', () => {
    clickerJSON('record start');
    clickerJSON(`go ${baseURL}/example`);

    // Stop first chunk
    const chunkPath1 = tmpPath('chunk1.zip');
    const chunk1 = clickerJSON(`record chunk stop -o "${chunkPath1}"`);
    assert.strictEqual(chunk1.ok, true, 'chunk stop should succeed');

    // Start second chunk
    clickerJSON('record chunk start');
    clickerJSON(`go ${baseURL}/example`);

    // Stop second chunk
    const chunkPath2 = tmpPath('chunk2.zip');
    clickerJSON(`record chunk stop -o "${chunkPath2}"`);

    // Verify both files
    assert.ok(fs.existsSync(chunkPath1), 'Chunk 1 should exist');
    assert.ok(fs.existsSync(chunkPath2), 'Chunk 2 should exist');

    const { tmpDir: td1, extractedDir: ed1 } = unzipRecording(chunkPath1);
    const { tmpDir: td2, extractedDir: ed2 } = unzipRecording(chunkPath2);
    tmpDirs.push(td1, td2);

    const events1 = readRecordingEvents(ed1);
    assert.ok(events1.length > 0, 'Chunk 1 should have events');
    const events2 = readRecordingEvents(ed2);
    assert.ok(events2.length > 0, 'Chunk 2 should have events');

    // Stop recording to clean up
    const stopPath = tmpPath('chunks-final.zip');
    clickerJSON(`record stop -o "${stopPath}"`);
  });

  test('record start --title sets trace viewer title', () => {
    clickerJSON('record start --title "CLI Title"');
    clickerJSON(`go ${baseURL}/example`);

    const zipPath = tmpPath('title.zip');
    clickerJSON(`record stop -o "${zipPath}"`);

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    tmpDirs.push(tmpDir);
    const events = readRecordingEvents(extractedDir);
    assert.strictEqual(events[0].title, 'CLI Title', 'First event should have the custom title');
  });

  test('recording with multiple actions captures multiple screenshots', () => {
    clickerJSON('record start --screenshots');
    clickerJSON(`go ${baseURL}/example`);

    // Inject interactive elements
    clicker('eval "document.body.innerHTML = \'<input id=inp type=text><button id=btn1>A</button><button id=btn2>B</button>\';"');
    clickerJSON('fill "#inp" "hello"');
    clickerJSON('click "#btn1"');
    clickerJSON('click "#btn2"');

    const zipPath = tmpPath('multi-screenshots.zip');
    clickerJSON(`record stop -o "${zipPath}"`);

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    tmpDirs.push(tmpDir);
    const events = readRecordingEvents(extractedDir);
    const frames = events.filter(e => e.type === 'screencast-frame');
    assert.ok(frames.length >= 3, `Should have >= 3 screencast-frame events (one per action), got ${frames.length}`);

    // Verify before/after events for each action
    const beforeEvents = events.filter(e => e.type === 'before' && e.method !== 'group');
    assert.ok(beforeEvents.length >= 3, `Should have >= 3 before events, got ${beforeEvents.length}`);

    // Check action titles map correctly
    const navBefore = beforeEvents.find(e => e.title === 'Page.navigate');
    assert.ok(navBefore, 'Should have before event with title Page.navigate');
    const fillBefore = beforeEvents.find(e => e.title === 'Element.fill');
    assert.ok(fillBefore, 'Should have before event with title Element.fill');
    const clickBefore = beforeEvents.find(e => e.title === 'Element.click');
    assert.ok(clickBefore, 'Should have before event with title Element.click');

    // Check input events with point and box for click/fill actions
    const inputEvents = events.filter(e => e.type === 'input');
    assert.ok(inputEvents.length > 0, 'Should have input events for element actions');
    assert.ok(inputEvents[0].point, 'input event should have point');
    assert.ok(typeof inputEvents[0].point.x === 'number', 'point.x should be a number');
    assert.ok(typeof inputEvents[0].point.y === 'number', 'point.y should be a number');
    assert.ok(inputEvents[0].box, 'input event should have box');
    assert.ok(inputEvents[0].box.width > 0, 'box.width should be > 0');
    assert.ok(inputEvents[0].box.height > 0, 'box.height should be > 0');
  });
});
