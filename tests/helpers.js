const fs = require('node:fs');
const path = require('node:path');
const EXE = process.platform === 'win32' ? '.exe' : '';
const VIBIUM = path.join(__dirname, '../clicker/bin/vibium') + EXE;
// The engine under test. CI runs each suite once per engine job by setting
// VIBIUM_ENGINE, the same parameter the Makefile threads through every
// engine-parity target as ENGINE. Suites read this instead of naming a
// browser: a new engine is a new job, not a new branch in the test.
const ENGINE = process.env.VIBIUM_ENGINE || 'chrome';

// The engine's release channel, same idea. Empty means the engine's default
// (release for Firefox). beta-watch.yml sets VIBIUM_ENGINE_CHANNEL=beta at the
// job level and runs the ordinary targets, so a suite that pins a channel
// itself both ignores that job and forces the other one to fetch a second
// browser it never installed.
const ENGINE_CHANNEL = process.env.VIBIUM_ENGINE_CHANNEL || '';

// The baked Chrome pin. Stable-channel resolution requires exactly this
// version (#579), so fixtures that seed a fake stable install must use it.
// Read from the source of truth rather than duplicated, so a pin bump
// cannot strand the fixtures.
function pinnedChromeVersion() {
  const pins = JSON.parse(fs.readFileSync(
    path.join(__dirname, '../clicker/internal/paths/browsers.json'), 'utf8'));
  if (!pins.chrome) throw new Error('browsers.json does not pin chrome');
  return pins.chrome;
}

module.exports = { VIBIUM, ENGINE, ENGINE_CHANNEL, pinnedChromeVersion };
