const path = require('node:path');
const EXE = process.platform === 'win32' ? '.exe' : '';
const VIBIUM = path.join(__dirname, '../clicker/bin/vibium') + EXE;
// The engine under test. CI runs each suite once per engine job by setting
// VIBIUM_ENGINE, the same parameter the Makefile threads through every
// engine-parity target as ENGINE. Suites read this instead of naming a
// browser: a new engine is a new job, not a new branch in the test.
const ENGINE = process.env.VIBIUM_ENGINE || 'chrome';

module.exports = { VIBIUM, ENGINE };
