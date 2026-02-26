/**
 * Parses annotated markdown tutorials and creates tests from code blocks.
 *
 * Annotations:
 *   <!-- helpers -->              — next code block defines shared helper functions
 *   <!-- server -->               — next code block defines an HTTP server handler
 *   <!-- test: async "name" -->   — next code block is an async test
 *   <!-- test: sync "name" -->    — next code block is a sync test
 *
 * Modes:
 *   Default — runner creates browser/page and passes `vibe` to each test.
 *   standalone: true — test code handles its own browser lifecycle.
 *       The runner only provides `assert`, `baseURL`, and `require`.
 *
 * Server and helpers can be passed directly via options (serverCode,
 * helpers) or parsed from the markdown.
 */

const { test, before, after } = require('node:test');
const assert = require('node:assert');
const { readFileSync } = require('fs');
const { resolve, join } = require('path');
const http = require('http');
const { fork } = require('child_process');

const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;

function extractBlocks(mdPath) {
  const fullPath = resolve(__dirname, '../../..', mdPath);
  const content = readFileSync(fullPath, 'utf8');
  const lines = content.split('\n');
  const blocks = [];

  let pending = null;
  let inCodeBlock = false;
  let isAnnotated = false;
  let codeLines = [];

  for (const line of lines) {
    if (!inCodeBlock) {
      const helpersMatch = line.match(/<!--\s*helpers\s*-->/);
      if (helpersMatch) {
        pending = { type: 'helpers' };
        continue;
      }

      const serverMatch = line.match(/<!--\s*server\s*-->/);
      if (serverMatch) {
        pending = { type: 'server' };
        continue;
      }

      const testMatch = line.match(/<!--\s*test:\s*(async|sync)\s+"([^"]+)"\s*-->/);
      if (testMatch) {
        pending = { type: 'test', mode: testMatch[1], name: testMatch[2] };
        continue;
      }

      if (line.match(/^```javascript\s*$/)) {
        inCodeBlock = true;
        if (pending) {
          isAnnotated = true;
          codeLines = [];
        } else {
          isAnnotated = false;
        }
        continue;
      }
    } else {
      if (line.match(/^```\s*$/)) {
        inCodeBlock = false;
        if (isAnnotated && pending) {
          blocks.push({ ...pending, code: codeLines.join('\n') });
          pending = null;
        }
        continue;
      }
      if (isAnnotated) {
        codeLines.push(line);
      }
    }
  }

  return blocks;
}

function runTutorial(mdPath, { browser, mode, serverCode, helpers: extraHelpers, standalone, requireFn }) {
  const _require = requireFn || require;
  const blocks = extractBlocks(mdPath);
  let helpers = extraHelpers || '';

  // Use explicit serverCode if provided, otherwise look in the markdown
  if (serverCode === undefined) {
    for (const block of blocks) {
      if (block.type === 'server') {
        serverCode = block.code;
        break;
      }
    }
  }

  // Server lifecycle — baseURL is set in the before() hook
  let baseURL = null;
  let _server = null;
  let _serverProcess = null;

  if (serverCode) {
    if (mode === 'async') {
      // Async: run server in-process
      before(async () => {
        const handler = new Function('req', 'res', serverCode);
        _server = http.createServer(handler);
        await new Promise((res) => {
          _server.listen(0, '127.0.0.1', () => {
            baseURL = `http://127.0.0.1:${_server.address().port}`;
            res();
          });
        });
      });
      after(() => { if (_server) _server.close(); });
    } else {
      // Sync: fork server into a child process (Atomics.wait blocks event loop)
      before(async () => {
        _serverProcess = fork(
          join(__dirname, 'tutorial-server-child.js'),
          [],
          { silent: true, env: { ...process.env, TUTORIAL_SERVER_CODE: serverCode } }
        );
        baseURL = await new Promise((resolve, reject) => {
          let data = '';
          _serverProcess.stdout.on('data', (chunk) => {
            data += chunk.toString();
            const line = data.trim().split('\n')[0];
            if (line.startsWith('http://')) resolve(line);
          });
          _serverProcess.on('error', reject);
          setTimeout(() => reject(new Error('Tutorial server startup timeout')), 5000);
        });
      });
      after(() => { if (_serverProcess) _serverProcess.kill(); });
    }
  }

  // Collect helpers from the markdown (they may appear anywhere in the file)
  for (const block of blocks) {
    if (block.type === 'helpers') helpers += block.code + '\n';
  }

  // Register tests
  for (const block of blocks) {
    if (block.type !== 'test' || block.mode !== mode) continue;

    if (standalone) {
      // Standalone: test code handles its own browser lifecycle
      if (mode === 'async') {
        test(block.name, async () => {
          const fn = new AsyncFunction('assert', 'baseURL', 'require', helpers + block.code);
          await fn(assert, baseURL, _require);
        });
      } else {
        test(block.name, () => {
          const fn = new Function('assert', 'baseURL', 'require', helpers + block.code);
          fn(assert, baseURL, _require);
        });
      }
    } else {
      // Default: runner manages browser lifecycle, passes vibe
      if (mode === 'async') {
        test(block.name, async () => {
          const bro = await browser.launch({ headless: true });
          try {
            const vibe = await bro.page();
            const fn = new AsyncFunction('vibe', 'assert', 'baseURL', 'require', helpers + block.code);
            await fn(vibe, assert, baseURL, require);
          } finally {
            await bro.close();
          }
        });
      } else {
        test(block.name, () => {
          const bro = browser.launch({ headless: true });
          try {
            const vibe = bro.page();
            const fn = new Function('vibe', 'assert', 'baseURL', 'require', helpers + block.code);
            fn(vibe, assert, baseURL, require);
          } finally {
            bro.close();
          }
        });
      }
    }
  }
}

module.exports = { runTutorial, extractBlocks };
