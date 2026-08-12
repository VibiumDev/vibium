'use strict';

const fs = require('node:fs');
const path = require('node:path');
const nodeTest = require('node:test');

const manifest = JSON.parse(
  fs.readFileSync(path.join(__dirname, '..', 'capabilities.json'), 'utf8')
);
const engine = process.env.VIBIUM_ENGINE || 'chrome';
if (!['chrome', 'firefox'].includes(engine)) {
  throw new Error(`unknown VIBIUM_ENGINE: ${engine}`);
}

const collectOnly = process.env.VIBIUM_CAPABILITY_COLLECT_ONLY === '1';
const audit = process.env.VIBIUM_CAPABILITY_AUDIT === '1';
const counts = { collected: 0, selected: 0, skipped: 0, capabilities: new Map() };
let summaryInstalled = false;
let summaryPrinted = false;

function validate(names) {
  for (const name of names) {
    if (!Object.hasOwn(manifest, name)) {
      throw new Error(`unknown capability: ${name}`);
    }
  }
}

function missingCapabilities(names) {
  validate(names);
  return names.filter((name) => !manifest[name].includes(engine));
}

function installSummary() {
  if (summaryInstalled) return;
  summaryInstalled = true;
  const printSummary = () => {
    if (summaryPrinted) return;
    summaryPrinted = true;
    console.log(
      `capabilities: engine=${engine} collected=${counts.collected} ` +
      `selected=${counts.selected} skipped=${counts.skipped}`
    );
    for (const [name, count] of [...counts.capabilities].sort()) {
      console.log(`capabilities: skip:${name}=${count}`);
    }
  };
  nodeTest.after(printSummary);
  process.on('beforeExit', printSummary);
}

function mergeOptions(args, skipReason) {
  // node:test's signature is ([name][, options][, fn]); the options object
  // must sit before the callback or node silently ignores it.
  const values = [...args];
  const fnIndex =
    values.length && typeof values[values.length - 1] === 'function' ? values.length - 1 : values.length;
  let optionsIndex = -1;
  for (let i = 0; i < fnIndex; i++) {
    if (typeof values[i] === 'object' && values[i] !== null) {
      optionsIndex = i;
      break;
    }
  }
  if (optionsIndex === -1) {
    optionsIndex = typeof values[0] === 'string' ? Math.min(1, fnIndex) : 0;
    values.splice(optionsIndex, 0, {});
  } else {
    values[optionsIndex] = { ...values[optionsIndex] };
  }
  if (skipReason) values[optionsIndex].skip = skipReason;
  if (collectOnly && fnIndex < values.length) values[values.length - 1] = () => {};
  return values;
}

function suite(...baseRequirements) {
  if (baseRequirements.length === 0) {
    throw new Error('cross-engine Node suites must declare at least one capability');
  }
  validate(baseRequirements);
  installSummary();
  let inherited = [...baseRequirements];

  function wrapTest(extraRequirements = []) {
    return (...args) => {
      const requirements = [...new Set([...inherited, ...extraRequirements])];
      const missing = missingCapabilities(requirements);
      counts.collected += 1;
      if (missing.length) {
        counts.skipped += 1;
        for (const name of missing) {
          counts.capabilities.set(name, (counts.capabilities.get(name) || 0) + 1);
        }
        // The manifest must not list an engine for a capability unless chrome
        // is also listed; empty entries are fine. Add an exemption mechanism
        // before introducing one.
        if (audit && engine === 'chrome') {
          const invalid = missing.filter((name) => manifest[name].length > 0);
          if (invalid.length) throw new Error(`Chrome audit rejected skips for: ${invalid.join(', ')}`);
        }
      } else {
        counts.selected += 1;
      }
      const reason = missing.length ? `${engine} lacks capabilities: ${missing.join(', ')}` : undefined;
      return nodeTest.test(...mergeOptions(args, reason));
    };
  }

  function wrapDescribe(extraRequirements = []) {
    return (...args) => {
      const callbackIndex = args.length - 1;
      const callback = args[callbackIndex];
      validate(extraRequirements);
      const values = mergeOptions(args, undefined);
      const mergedCallbackIndex = callbackIndex + (values.length - args.length);
      values[mergedCallbackIndex] = (...callbackArgs) => {
        const previous = inherited;
        inherited = [...new Set([...inherited, ...extraRequirements])];
        try {
          const result = callback(...callbackArgs);
          if (result && typeof result.then === 'function') {
            // Requirements are restored synchronously below; tests registered
            // after an await would silently lose them.
            throw new Error('cross-engine describe callbacks must be synchronous');
          }
          return result;
        } finally {
          inherited = previous;
        }
      };
      return nodeTest.describe(...values);
    };
  }

  const test = wrapTest();
  const describe = wrapDescribe();
  test.requires = (...names) => wrapTest(names);
  describe.requires = (...names) => wrapDescribe(names);

  const noopHook = collectOnly ? () => {} : null;
  return {
    test,
    describe,
    before: noopHook || nodeTest.before,
    after: noopHook || nodeTest.after,
    beforeEach: noopHook || nodeTest.beforeEach,
    afterEach: noopHook || nodeTest.afterEach,
  };
}

module.exports = { suite, manifest, engine };
