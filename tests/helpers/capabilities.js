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
  const values = [...args];
  const options = typeof values[1] === 'object' && values[1] !== null ? { ...values[1] } : {};
  if (skipReason) options.skip = skipReason;
  if (typeof values[1] === 'object' && values[1] !== null) values[1] = options;
  else values.splice(1, 0, options);
  if (collectOnly) values[values.length - 1] = () => {};
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
          return callback(...callbackArgs);
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
