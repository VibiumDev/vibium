/**
 * Minimal HTTP server for sync tutorial tests.
 * Runs in a child process because the sync API blocks the event loop.
 *
 * Receives the request handler function body via TUTORIAL_SERVER_CODE env var.
 * Prints the base URL to stdout once listening.
 */
const http = require('http');

const handler = new Function('req', 'res', process.env.TUTORIAL_SERVER_CODE);
const server = http.createServer(handler);

server.listen(0, '127.0.0.1', () => {
  const { port } = server.address();
  process.stdout.write(`http://127.0.0.1:${port}\n`);
});
