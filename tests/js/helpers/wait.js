/**
 * Race a promise against a timeout. If the timeout fires first, rejects with
 * an Error whose message identifies what we were waiting for.
 *
 * Use this in tests to replace fixed `await vibe.wait(N)` sleeps that were
 * really waiting for an event to fire. A real bug still fails fast (within
 * `timeoutMs`); a healthy run finishes as soon as the event arrives.
 */
function withTimeout(promise, timeoutMs, label) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(
      () => reject(new Error(`timed out after ${timeoutMs}ms waiting for: ${label}`)),
      timeoutMs,
    );
  });
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer));
}

module.exports = { withTimeout };
