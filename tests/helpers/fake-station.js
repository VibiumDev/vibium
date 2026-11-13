/**
 * A stand-in for a Vibium Station's Record Store, run in its own process so
 * a test that blocks on execSync still gets answered. Prints its base URL
 * once it listens.
 *
 *   POST /v1/records/upload-auth   needs Authorization: Bearer <KEY>; answers
 *                                  with a fixed authorization and the upload
 *                                  URL; refuses a file named refused.zip
 *                                  with 413
 *   PUT  /upload                   needs that authorization; checks the body
 *                                  hashes to what was authorized; answers
 *                                  with a blob descriptor
 *   GET  /seen                     what the last two requests carried
 *   POST /reset                    forgets them
 */

const http = require('node:http');
const crypto = require('node:crypto');

const KEY = process.env.FAKE_STATION_KEY || 'stk_test_key';
const AUTH = 'Nostr dGVzdA==';
let seen = { auth: null, put: null };
let base = '';

function json(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}

const server = http.createServer((req, res) => {
  const chunks = [];
  req.on('data', (c) => chunks.push(c));
  req.on('end', () => {
    const body = Buffer.concat(chunks);
    if (req.method === 'GET' && req.url === '/seen') return json(res, 200, seen);
    if (req.method === 'POST' && req.url === '/reset') {
      seen = { auth: null, put: null };
      return json(res, 200, {});
    }
    if (req.method === 'POST' && req.url === '/v1/records/upload-auth') {
      const parsed = JSON.parse(body.toString() || '{}');
      seen.auth = { authorization: req.headers.authorization, body: parsed };
      if (req.headers.authorization !== `Bearer ${KEY}`) {
        return json(res, 401, { error: 'sign in, or send Authorization: Bearer <token>' });
      }
      if (parsed.name === 'refused.zip') {
        return json(res, 413, { error: 'a file is at most 4 bytes here' });
      }
      return json(res, 200, { authorization: AUTH, upload: `${base}/upload`, expires: '2026-11-06T00:00:00Z' });
    }
    if (req.method === 'PUT' && req.url === '/upload') {
      const sha256 = crypto.createHash('sha256').update(body).digest('hex');
      seen.put = { authorization: req.headers.authorization, type: req.headers['content-type'], size: body.length, sha256 };
      if (req.headers.authorization !== AUTH) {
        return json(res, 401, { error: 'send Authorization: Nostr <base64 event>' });
      }
      if (!seen.auth || sha256 !== seen.auth.body.sha256) {
        return json(res, 400, { error: 'the body does not hash to the authorized file' });
      }
      return json(res, 200, { url: `${base}/${sha256}.zip`, sha256, size: body.length, type: 'application/zip', uploaded: 1,
        name: seen.auth.body.name, uploader: 'token:test', hold: 'hld_test', device: 'rig_1/phone', event: 'ev1' });
    }
    json(res, 404, { error: 'not here' });
  });
});

server.listen(0, '127.0.0.1', () => {
  base = `http://127.0.0.1:${server.address().port}`;
  console.log(base);
});
