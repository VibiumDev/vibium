const assert = require('node:assert/strict');
const { spawn } = require('node:child_process');
const { createInterface } = require('node:readline');
const { once } = require('node:events');
(async () => {
  const proc = spawn(process.env.VIBIUM_BIN_PATH, ['mcp', '--headless'], { stdio: ['pipe', 'pipe', 'pipe'] });
  proc.stderr.resume();
  const waiting = new Map(); let id = 0;
  createInterface({ input: proc.stdout }).on('line', line => {
    const message = JSON.parse(line); const cb = waiting.get(message.id);
    if (cb) { waiting.delete(message.id); cb(message); }
  });
  const call = (method, params) => new Promise(resolve => {
    waiting.set(++id, resolve); proc.stdin.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n');
  });
  const tool = async (name, args) => {
    const response = await call('tools/call', { name, arguments: args });
    assert.ok(!response.error, JSON.stringify(response));
    assert.ok(!response.result.isError, JSON.stringify(response));
    return response.result.content.filter(c => c.type === 'text').map(c => c.text).join('\n');
  };
  try {
    await call('initialize', { capabilities: {} });
    const listing = await call('tools/list', {});
    const schema = listing.result.tools.find(t => t.name === 'vibium_verify');
    assert.ok(schema.inputSchema.properties.record && schema.inputSchema.properties.page);
    const invalid = await call('tools/call', { name: 'vibium_verify', arguments: { claim: 'claim', record: process.env.VERIFY_INPUT, page: 'invalid-page' } });
    assert.ok(invalid.error || invalid.result.isError);
    if (process.env.VERIFY_ARCHIVE_ONLY !== '1') {
      const created = await tool('browser_new_page', { url: process.env.VERIFY_URL });
      const page = created.match(/\(page: ([^)]+)\)/)[1];
      await tool('browser_new_page', { url: process.env.VERIFY_URL + '/other' });
      await tool('browser_record_start', { video: false, path: process.env.VERIFY_OUTPUT });
      const result = JSON.parse(await tool('vibium_verify', { claim: 'changing my display name persists after refresh', page }));
      assert.equal(result.status, 'passed');
      await tool('browser_record_stop', {});
    }
    const result = JSON.parse(await tool('vibium_verify', { claim: 'archive evidence', record: process.env.VERIFY_INPUT }));
    assert.equal(result.status, 'inconclusive');
  } finally {
    proc.stdin.end();
    await once(proc, 'exit');
  }
})().catch(e => { console.error(e); process.exitCode = 1; });
