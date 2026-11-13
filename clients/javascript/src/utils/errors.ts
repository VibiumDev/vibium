/**
 * Custom error types for the Vibium client library.
 */

/**
 * ConnectionError is thrown when connecting to the browser fails.
 */
export class ConnectionError extends Error {
  constructor(
    public url: string,
    public cause?: Error
  ) {
    super(cause ? `Failed to connect to ${url}: ${cause.message}` : `Failed to connect to ${url}`);
    this.name = 'ConnectionError';
  }
}

/**
 * TimeoutError is thrown when a wait operation times out.
 *
 * Constructed from a selector/timeout pair, or from a single complete
 * message for timeouts the engine reports.
 */
export class TimeoutError extends Error {
  public timeout: number;
  constructor(
    public selector: string,
    timeout?: number,
    public reason?: string
  ) {
    super(
      timeout === undefined
        ? selector
        : reason
          ? `Timeout after ${timeout}ms waiting for '${selector}': ${reason}`
          : `Timeout after ${timeout}ms waiting for '${selector}'`
    );
    this.timeout = timeout ?? 0;
    this.name = 'TimeoutError';
  }
}

/**
 * ElementNotFoundError is thrown when a selector matches no elements.
 */
export class ElementNotFoundError extends Error {
  constructor(public selector: string) {
    super(`Element not found: ${selector}`);
    this.name = 'ElementNotFoundError';
  }
}

/**
 * BiDiError is thrown when the engine reports a command failure with no more
 * specific mapping. Mirrors the Python client's BiDiError.
 */
export class BiDiError extends Error {
  constructor(
    public error: string,
    message: string
  ) {
    super(`${error}: ${message}`);
    this.name = 'BiDiError';
  }
}

/**
 * SpeechError is thrown when say or listen fails. `code` names the failure,
 * such as 'no_speech' or 'not_configured'.
 */
export class SpeechError extends Error {
  constructor(
    public code: string,
    message: string
  ) {
    super(message);
    this.name = 'SpeechError';
  }
}

/**
 * Maps an engine error response onto the exported error classes, the same
 * way the Python client does: element-not-found failures and engine-reported
 * timeouts get their dedicated classes, everything else is a BiDiError.
 */
export function errorFromResponse(code: string, message: string): Error {
  if (message.includes('element not found')) {
    return new ElementNotFoundError(message);
  }
  if (code === 'timeout') {
    return new TimeoutError(message);
  }
  return new BiDiError(code, message);
}

/**
 * BrowserCrashedError is thrown when the browser process dies unexpectedly.
 */
export class BrowserCrashedError extends Error {
  constructor(
    public exitCode: number,
    public output?: string
  ) {
    const msg = output
      ? `Browser crashed with exit code ${exitCode}: ${output}`
      : `Browser crashed with exit code ${exitCode}`;
    super(msg);
    this.name = 'BrowserCrashedError';
  }
}
