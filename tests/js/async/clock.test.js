/**
 * JS Library Tests: Clock
 * Tests page.clock.install, fastForward, runFor, pauseAt, resume, setFixedTime, setSystemTime.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../clients/javascript/dist');

// --- Local test server ---

let server;
let baseURL;

const HTML_PAGE = `
<html>
<head><title>Clock Test</title></head>
<body>
  <div id="output"></div>
  <script>
    // Page sets up nothing by default — tests install the fake clock
  </script>
</body>
</html>
`;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(HTML_PAGE);
  });

  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      baseURL = `http://127.0.0.1:${port}`;
      resolve();
    });
  });
});

after(() => {
  if (server) server.close();
});

// --- Tests ---

describe('Clock: install', () => {
  test('install() overrides Date.now()', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      // Get real time before install
      const realTime = await vibe.eval('Date.now()');
      assert.ok(typeof realTime === 'number' && realTime > 0);

      await vibe.clock.install();

      // After install, Date.now() should be frozen at the install time
      const t1 = await vibe.eval('Date.now()');
      const t2 = await vibe.eval('Date.now()');
      assert.strictEqual(t1, t2, 'Date.now() should return the same value when clock is installed');
    } finally {
      await bro.close();
    }
  });

  test('install({time}) sets initial time', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const targetTime = new Date('2025-01-01T00:00:00Z').getTime();
      await vibe.clock.install({ time: targetTime });

      const now = await vibe.eval('Date.now()');
      assert.strictEqual(now, targetTime, `Date.now() should be ${targetTime}, got ${now}`);
    } finally {
      await bro.close();
    }
  });

  test('install() with Date object', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.clock.install({ time: new Date('2025-06-15T12:00:00Z') });

      const year = await vibe.eval('new Date().getFullYear()');
      assert.strictEqual(year, 2025, `Year should be 2025, got ${year}`);
    } finally {
      await bro.close();
    }
  });

  test('install() with ISO string', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.clock.install({ time: '2030-12-25T00:00:00Z' });

      const year = await vibe.eval('new Date().getFullYear()');
      assert.strictEqual(year, 2030, `Year should be 2030, got ${year}`);
    } finally {
      await bro.close();
    }
  });

  test('double install is safe (idempotent)', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.clock.install({ time: 1000000 });
      const t1 = await vibe.eval('Date.now()');

      // Install again — should not throw or reset
      await vibe.clock.install();
      const t2 = await vibe.eval('Date.now()');

      assert.strictEqual(t1, t2, 'Second install should not change the time');
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: setFixedTime', () => {
  test('setFixedTime() freezes Date.now()', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      const fixedMs = new Date('2025-01-01T00:00:00Z').getTime();
      await vibe.clock.setFixedTime(fixedMs);

      const now1 = await vibe.eval('Date.now()');
      const now2 = await vibe.eval('Date.now()');
      assert.strictEqual(now1, fixedMs, `Date.now() should be frozen at ${fixedMs}`);
      assert.strictEqual(now2, fixedMs, 'Date.now() should still be frozen');
    } finally {
      await bro.close();
    }
  });

  test('setFixedTime() with Date object', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      await vibe.clock.setFixedTime(new Date('2025-06-15T12:00:00Z'));

      const year = await vibe.eval('new Date().getUTCFullYear()');
      assert.strictEqual(year, 2025);
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: fastForward', () => {
  test('fastForward() fires setTimeout callbacks', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      // Set up a timeout that writes to #output
      await vibe.eval(`
        window.setTimeout(() => {
          document.getElementById('output').textContent = 'timer fired';
        }, 5000)
      `);

      // Verify it hasn't fired yet
      const before = await vibe.eval('document.getElementById("output").textContent');
      assert.strictEqual(before, '', 'Timer should not have fired yet');

      // Fast forward past the timer
      await vibe.clock.fastForward(6000);

      const after = await vibe.eval('document.getElementById("output").textContent');
      assert.strictEqual(after, 'timer fired', 'Timer should have fired after fast-forward');
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: runFor', () => {
  test('runFor() fires all callbacks including intervals', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      // Set up an interval that increments a counter
      await vibe.eval(`
        window.__counter = 0;
        window.setInterval(() => { window.__counter++; }, 100)
      `);

      // Run for 500ms — should fire interval ~5 times
      await vibe.clock.runFor(500);

      const count = await vibe.eval('window.__counter');
      assert.strictEqual(count, 5, `Counter should be 5, got ${count}`);
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: pauseAt', () => {
  test('pauseAt() stops timer execution', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      const targetTime = await vibe.eval('Date.now()') + 10000;

      await vibe.eval(`
        window.__timerFired = false;
        window.setTimeout(() => { window.__timerFired = true; }, 20000)
      `);

      await vibe.clock.pauseAt(targetTime);

      // Timer at +20000 shouldn't fire (we only jumped to +10000)
      const fired = await vibe.eval('window.__timerFired');
      assert.strictEqual(fired, false, 'Timer should not fire — paused before its time');

      // Time should be frozen at the paused value
      const now = await vibe.eval('Date.now()');
      assert.strictEqual(now, targetTime, `Date.now() should be ${targetTime}`);
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: resume', () => {
  test('resume() resumes real-time flow', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      const startTime = await vibe.eval('Date.now()');

      // Pause, then resume
      await vibe.clock.pauseAt(startTime);
      await vibe.clock.resume();

      // Wait a bit of real time
      await vibe.wait(200);

      const afterTime = await vibe.eval('Date.now()');
      assert.ok(afterTime > startTime, `Time should have advanced after resume (${afterTime} > ${startTime})`);
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: setSystemTime', () => {
  test('setSystemTime() changes time without firing timers', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      await vibe.eval(`
        window.__timerFired = false;
        window.setTimeout(() => { window.__timerFired = true; }, 1000)
      `);

      // Jump far ahead — but setSystemTime should NOT fire timers
      const futureTime = Date.now() + 100000;
      await vibe.clock.setSystemTime(futureTime);

      const fired = await vibe.eval('window.__timerFired');
      assert.strictEqual(fired, false, 'Timer should NOT fire from setSystemTime');

      const now = await vibe.eval('Date.now()');
      assert.strictEqual(now, futureTime, `Date.now() should be ${futureTime}`);
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: survives navigation', () => {
  test('clock persists after page.go() to a new URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await vibe.clock.install();

      // Verify clock is installed
      const before = await vibe.eval('typeof window.__vibiumClock');
      assert.strictEqual(before, 'object', 'Clock should be installed before navigation');

      // Navigate to a different page (same origin, different path)
      await vibe.go(baseURL + '/?after');

      // Clock should auto-reinstall via preload script
      const after = await vibe.eval('typeof window.__vibiumClock');
      assert.strictEqual(after, 'object', 'Clock should persist after navigation');

      // Verify clock still works
      await vibe.clock.setFixedTime(new Date('2025-01-01T00:00:00Z').getTime());
      const year = await vibe.eval('new Date().getUTCFullYear()');
      assert.strictEqual(year, 2025, 'Clock should be functional after navigation');
    } finally {
      await bro.close();
    }
  });
});

describe('Clock: setTimezone', () => {
  test('install({timezone}) overrides Intl timezone', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.clock.install({ timezone: 'America/New_York' });

      const tz = await vibe.eval('Intl.DateTimeFormat().resolvedOptions().timeZone');
      assert.strictEqual(tz, 'America/New_York', `Timezone should be America/New_York, got ${tz}`);
    } finally {
      await bro.close();
    }
  });

  test('setTimezone() changes timezone independently', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.clock.setTimezone('Asia/Tokyo');

      const tz = await vibe.eval('Intl.DateTimeFormat().resolvedOptions().timeZone');
      assert.strictEqual(tz, 'Asia/Tokyo', `Timezone should be Asia/Tokyo, got ${tz}`);
    } finally {
      await bro.close();
    }
  });

  test('setTimezone("") resets to system default', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      // Set a non-default timezone
      await vibe.clock.setTimezone('Pacific/Honolulu');
      const tz1 = await vibe.eval('Intl.DateTimeFormat().resolvedOptions().timeZone');
      assert.strictEqual(tz1, 'Pacific/Honolulu');

      // Reset to system default
      await vibe.clock.setTimezone('');
      const tz2 = await vibe.eval('Intl.DateTimeFormat().resolvedOptions().timeZone');
      assert.notStrictEqual(tz2, 'Pacific/Honolulu', 'Timezone should have been reset');
    } finally {
      await bro.close();
    }
  });

  test('timezone + time work together', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      // Install clock at midnight UTC Jan 1, 2025 with Tokyo timezone (UTC+9)
      await vibe.clock.install({
        time: new Date('2025-01-01T00:00:00Z').getTime(),
        timezone: 'Asia/Tokyo',
      });

      // In Tokyo, midnight UTC is 9:00 AM
      const hour = await vibe.eval('new Date().getHours()');
      assert.strictEqual(hour, 9, `Hour in Tokyo should be 9, got ${hour}`);
    } finally {
      await bro.close();
    }
  });
});
