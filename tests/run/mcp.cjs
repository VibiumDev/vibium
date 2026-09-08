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
    const schema = listing.result.tools.find(t => t.name === 'vibium_run');
    assert.ok(schema.inputSchema.properties.goal && schema.inputSchema.properties.page);
    assert.equal(schema.inputSchema.properties.record, undefined);
    const invalid = await call('tools/call', { name: 'vibium_run', arguments: { goal: 'goal', record: 'record.zip' } });
    assert.ok(invalid.error || invalid.result.isError);
    const created = await tool('browser_new_page', { url: process.env.RUN_TEST_URL });
    const page = created.match(/\(page: ([^)]+)\)/)[1];
    await tool('browser_new_page', { url: process.env.RUN_TEST_URL + '/other' });
    await tool('browser_record_start', { video: false, path: process.env.RUN_TEST_OUTPUT });
    const result = JSON.parse(await tool('vibium_run', { goal: 'change name', page }));
    assert.equal(result.status, 'completed'); assert.equal(result.goal, 'change name');
    assert.equal(JSON.parse(await tool('vibium_check', { claim: 'the name persisted', page })).status, 'passed');
    const overrides = { provider: 'google', model: 'run-model', baseURL: process.env.VIBIUM_AI_BASE_URL, reasoningEffort: '' };
    assert.equal(JSON.parse(await tool('vibium_run', { goal: 'change name', page, ...overrides })).status, 'completed');
    assert.equal(JSON.parse(await tool('vibium_check', { claim: 'the name persisted', page, ...overrides, model: 'check-model' })).status, 'passed');
    assert.equal(JSON.parse(await tool('vibium_run', { goal: 'not possible' })).status, 'not_completed');
    await tool('browser_set', { selector: '#consent', page });
    assert.equal(await tool('browser_is_set', { selector: '#consent', page }), 'true');
    await tool('browser_set', { selector: '#consent', value: false, page });
    assert.equal(await tool('browser_is_set', { selector: '#consent', page }), 'false');
    await tool('browser_unset', { selector: '#consent', page });
    await tool('browser_record_stop', {});
  } finally {
    proc.stdin.end();
    await once(proc, 'exit');
  }
})().catch(e => { console.error(e); process.exitCode = 1; });
