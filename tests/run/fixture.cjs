const http = require('node:http');
const assert = require('node:assert/strict');
const listen = server => new Promise(r => server.listen(0, '127.0.0.1', () => r(`http://127.0.0.1:${server.address().port}`)));

function decode(body, url) {
  if (body.contents) {
    const messages = body.contents;
    return { family: 'google', model: decodeURIComponent(url.match(/models\/(.*):generateContent/)[1]),
      system: body.systemInstruction.parts.map(p => p.text || '').join(''),
      input: messages[0].parts[0].text,
      tools: body.tools[0].functionDeclarations.map(t => t.name),
      observations: messages.flatMap(m => m.parts.filter(p => p.functionResponse).map(p => p.functionResponse.response.output)),
      ping: messages.flatMap(m => m.parts.filter(p => p.functionResponse?.name === 'verifier_ping').map(p => p.functionResponse.response.output)),
    };
  }
  if (body.system) {
    return { family: 'anthropic', model: body.model, system: body.system,
      input: body.messages[0].content[0].text, tools: body.tools.map(t => t.name),
      observations: body.messages.flatMap(m => m.content.filter(p => p.type === 'tool_result').map(p => p.content)),
      ping: body.tools[0].name === 'verifier_ping' ? body.messages.flatMap(m => m.content.filter(p => p.type === 'tool_result').map(p => p.content)) : [],
    };
  }
  return { family: 'openai', model: body.model, system: body.messages[0].content,
    input: body.messages[1].content, tools: body.tools.map(t => t.function.name),
    observations: body.messages.filter(m => m.role === 'tool').map(m => m.content),
    ping: body.tools[0].function.name === 'verifier_ping' ? body.messages.filter(m => m.role === 'tool').map(m => m.content) : [],
  };
}
function answer(res, family, step, result, index) {
  const text = JSON.stringify(result);
  // Real Claude wraps the final verdict in a markdown fence behind a line of
  // prose despite being told to return only JSON; the openai family answers
  // bare. Each emulation matches its provider's observed behavior so parser
  // regressions fail here instead of on the first live call (#506, #514).
  const fenced = 'The evidence is clear. Here is the result:\n\n```json\n' + text + '\n```';
  res.setHeader('Content-Type', 'application/json');
  if (family === 'anthropic') {
    res.end(JSON.stringify({ stop_reason: step ? 'tool_use' : 'end_turn', content: step ? [
      { type: 'text', text: 'PRIVATE-REASONING' }, { type: 'tool_use', id: `tool-${index}`, name: step[0], input: step[1] },
    ] : [{ type: 'text', text: fenced }] }));
  } else if (family === 'google') {
    res.end(JSON.stringify({ candidates: [{ finishReason: 'STOP', content: { role: 'model', parts: step ? [
      { thought: true, text: 'PRIVATE-REASONING' }, { functionCall: { name: step[0], args: step[1], id: `tool-${index}` }, thoughtSignature: `opaque-${index}` },
    ] : [{ text }] } }] }));
  } else {
    res.end(JSON.stringify({ choices: [{ finish_reason: step ? 'tool_calls' : 'stop', message: { role: 'assistant', content: step ? 'PRIVATE-REASONING' : text,
      ...(step ? { tool_calls: [{ id: `tool-${index}`, type: 'function', function: { name: step[0], arguments: JSON.stringify(step[1]) } }] } : {}) } }] }));
  }
}
async function fixture() {
  const errors = [], requests = [];
  const app = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'text/html');
    if (req.url === '/other') return res.end('<h1>Other page</h1><input id="name" value="Other">');
    res.end(`<!doctype html><title>Account</title><h1>Account</h1><form>
      <label>Display name<input id="name"></label><button>Save</button></form><p role="status" id="status"></p><label>Consent<input type="checkbox" id="consent"></label>
      <script>const input=document.querySelector('#name');input.value=localStorage.getItem('name')||'Original';
      document.querySelector('form').onsubmit=e=>{e.preventDefault();localStorage.setItem('name',input.value);document.querySelector('#status').textContent='Saved '+input.value;console.log('Saved')};</script>`);
  });
  const url = await listen(app);
  const provider = http.createServer(async (req, res) => {
    try {
      let raw = ''; for await (const c of req) raw += c;
      assert.ok(!raw.includes('PRIVATE-REASONING'));
      const body = JSON.parse(raw), d = decode(body, req.url);
      requests.push(d);
      if (d.family === 'anthropic') { assert.equal(req.headers['x-api-key'], 'native-key'); assert.equal(req.headers['anthropic-version'], '2023-06-01'); }
      if (d.family === 'google') assert.equal(req.headers['x-goog-api-key'], 'native-key');
      if (d.model === 'error-model') { res.writeHead(401); return res.end('{"error":{"message":"SECRET-ERROR-BODY"}}'); }
      if (d.tools[0] === 'verifier_ping') return answer(res, d.family, d.ping.length ? null : ['verifier_ping', {}], d.ping.length ? JSON.parse(d.ping.at(-1)) : null, 0);
      if (d.tools.every(t => t.startsWith('trace_'))) {
        assert.ok(d.model);
        return answer(res, d.family, null, { status: 'inconclusive', summary: 'Archive does not establish the claim.', evidence: [] }, 0);
      }
      assert.ok(!d.tools.some(t => /eval|stop|launch|record|^vibium_(run|check)$/.test(t)), 'operation obtained unrestricted or recursive tools');
      const isRun = d.system.includes('accomplish');
      assert.ok(d.system.includes(isRun ? 'accomplish' : 'independent software verifier'));
      const obs = d.observations;
      let steps;
      if (d.input === 'not possible') steps = [];
      else if (d.input === 'credential test') steps = [['browser_fill', { selector: '#password', text: 'TEST-PASSWORD-SECRET' }], ['browser_get_value', { selector: '#password' }]];
      else steps = isRun ? [
        ['browser_navigate', { url }], ['browser_fill', { selector: '#name', text: 'Updated' }], ['browser_click', { selector: 'button' }], ['browser_get_value', { selector: '#name' }], ['browser_screenshot', {}],
      ] : [['browser_reload', {}], ['browser_get_value', { selector: '#name' }]];
      const step = steps[obs.length];
      if (!step && d.input !== 'not possible') {
        if (d.input === 'credential test') assert.match(obs[1], /Password fields are unavailable/);
        else assert.ok(obs[isRun ? 3 : 1].includes('Updated'), 'goal not accomplished in browser');
      }
      const result = { status: isRun ? (d.input === 'not possible' ? 'not_completed' : 'completed') : 'passed', summary: 'Fixture observed the outcome.', evidence: [{ type: 'observation', summary: 'Observed the requested state.' }] };
      answer(res, d.family, step, result, obs.length);
    } catch (err) { errors.push(err.stack); res.writeHead(500); res.end('{}'); }
  });
  const endpoint = await listen(provider);
  return { url, endpoint, errors, requests,
    env: { VIBIUM_AI_PROVIDER: 'openai-compatible', VIBIUM_AI_MODEL: 'check-model', VIBIUM_AI_BASE_URL: `${endpoint}/v1`, VIBIUM_AI_REASONING_EFFORT: '', OPENAI_API_KEY: '', ANTHROPIC_API_KEY: 'native-key', GOOGLE_API_KEY: 'native-key' },
    close: async () => { for (const s of [provider, app]) { s.closeAllConnections(); await new Promise(r => s.close(r)); } assert.deepEqual(errors, []); },
  };
}
module.exports = { fixture };
