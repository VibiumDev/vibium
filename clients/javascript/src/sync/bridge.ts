import { Worker, MessageChannel, receiveMessageOnPort, MessagePort } from 'worker_threads';
import * as path from 'path';

interface CommandResult {
  result?: unknown;
  error?: string;
}

interface CallbackRequest {
  handlerId: string;
  data: unknown;
}

// Track active bridges for cleanup on process exit
const activeBridges: Set<SyncBridge> = new Set();

function cleanup() {
  for (const bridge of activeBridges) {
    try {
      // Try graceful quit first (with short timeout)
      bridge.tryQuit();
    } catch {
      // Ignore errors during cleanup
    }
  }
  activeBridges.clear();
}

// Register cleanup handlers once
let handlersRegistered = false;
function registerCleanupHandlers() {
  if (handlersRegistered) return;
  handlersRegistered = true;

  process.on('exit', cleanup);
  process.on('SIGINT', () => {
    cleanup();
    process.exit(130);
  });
  process.on('SIGTERM', () => {
    cleanup();
    process.exit(143);
  });
}

/**
 * Signal protocol (2-slot SharedArrayBuffer):
 *   signal[0]: worker → main  (0=idle, 1=result ready, 2=callback needed)
 *   signal[1]: main → worker  (0=idle, 1=callback response ready)
 */
export class SyncBridge {
  private worker: Worker;
  private signal: Int32Array;
  private commandId = 0;
  private terminated = false;

  // Callback infrastructure
  private callbackPortMain: MessagePort;
  private callbackPortWorker: MessagePort;
  private handlers = new Map<string, Function>();

  private constructor(
    worker: Worker,
    signal: Int32Array,
    callbackPortMain: MessagePort,
    callbackPortWorker: MessagePort,
  ) {
    this.worker = worker;
    this.signal = signal;
    this.callbackPortMain = callbackPortMain;
    this.callbackPortWorker = callbackPortWorker;
  }

  static create(): SyncBridge {
    registerCleanupHandlers();

    // 2 slots: signal[0] = worker→main, signal[1] = main→worker
    const signal = new Int32Array(new SharedArrayBuffer(8));

    // Persistent callback channel
    const { port1: callbackPortMain, port2: callbackPortWorker } = new MessageChannel();

    // Resolve worker path - works for both dev and built scenarios
    const workerPath = path.join(__dirname, 'worker.js');

    const worker = new Worker(workerPath, {
      workerData: { signal, callbackPort: callbackPortWorker },
      transferList: [callbackPortWorker],
    });

    const bridge = new SyncBridge(worker, signal, callbackPortMain, callbackPortWorker);
    activeBridges.add(bridge);

    return bridge;
  }

  /** Register a callback handler that can be invoked from the worker thread. */
  registerHandler(id: string, handler: Function): void {
    this.handlers.set(id, handler);
  }

  /** Remove a previously registered callback handler. */
  unregisterHandler(id: string): void {
    this.handlers.delete(id);
  }

  /** Process any pending callbacks that fired between bridge calls. */
  private processPendingCallbacks(): void {
    while (Atomics.load(this.signal, 0) === 2) {
      this.handleCallback();
    }
  }

  /** Handle a single callback request from the worker. */
  private handleCallback(): void {
    // Spin-wait for the port message — the worker posted it before setting
    // signal[0]=2, but postMessage is async so it may arrive slightly after
    // the Atomics signal. Limit iterations to avoid hanging if worker dies.
    const maxSpinIterations = 60_000; // ~60s with 1ms sleeps
    let cbMsg = receiveMessageOnPort(this.callbackPortMain);
    let spinCount = 0;
    while (!cbMsg) {
      spinCount++;
      if (spinCount >= maxSpinIterations) {
        throw new Error('Timed out waiting for callback message from worker (60s)');
      }
      // Brief sleep to avoid busy-spinning — use Atomics.wait on a dummy array
      Atomics.wait(this.signal, 1, Atomics.load(this.signal, 1), 1);
      cbMsg = receiveMessageOnPort(this.callbackPortMain);
    }

    let decision: unknown = null;
    const req = cbMsg.message as CallbackRequest;
    const handler = this.handlers.get(req.handlerId);
    if (handler) {
      try {
        decision = handler(req.data);
      } catch {
        // Handler errors produce default decision (null)
      }
    }

    // Reset signal[0] BEFORE posting the decision to avoid a race condition:
    // If we reset after postMessage, the worker might receive the decision,
    // prepare the next callback, and set signal[0]=2 before our reset —
    // overwriting it back to 0 and causing a deadlock.
    Atomics.store(this.signal, 0, 0);

    // Post decision back to worker via port
    this.callbackPortMain.postMessage({ decision });
  }

  call<T = unknown>(method: string, args: unknown[] = [], commandTimeoutMs = 60_000): T {
    // Handle any callbacks that fired between bridge calls
    this.processPendingCallbacks();

    const cmd = { id: this.commandId++, method, args };

    // Create a channel for this call's result
    const { port1, port2 } = new MessageChannel();

    // Reset both signals
    Atomics.store(this.signal, 0, 0);
    Atomics.store(this.signal, 1, 0);

    // Send command with the port
    this.worker.postMessage({ cmd, port: port2 }, [port2]);

    // Block until worker signals — loop to handle mid-command callbacks
    // Use a 1000ms Atomics.wait timeout and track total elapsed time (60s max)
    const waitSliceMs = 1000;
    const startTime = Date.now();

    for (;;) {
      const waitResult = Atomics.wait(this.signal, 0, 0, waitSliceMs);

      if (waitResult === 'timed-out') {
        // Check total elapsed time
        if (Date.now() - startTime >= commandTimeoutMs) {
          port1.close();
          throw new Error(`Bridge call '${method}' timed out after ${commandTimeoutMs / 1000}s — worker may have died`);
        }
        // Worker might still be alive, keep waiting
        continue;
      }

      const sig = Atomics.load(this.signal, 0);

      if (sig === 1) {
        // Result ready — read and return
        const message = receiveMessageOnPort(port1);
        port1.close();

        if (!message) {
          throw new Error('No response from worker');
        }

        const response = message.message as CommandResult;
        if (response.error) {
          throw new Error(response.error);
        }
        return response.result as T;
      }

      if (sig === 2) {
        this.handleCallback();
      }
    }
  }

  tryQuit(): void {
    if (this.terminated) return;

    try {
      // Try to send quit command with timeout for graceful shutdown
      const cmd = { id: this.commandId++, method: 'quit', args: [] };
      const { port1, port2 } = new MessageChannel();

      Atomics.store(this.signal, 0, 0);
      Atomics.store(this.signal, 1, 0);
      this.worker.postMessage({ cmd, port: port2 }, [port2]);

      // Wait with timeout (5s for cleanup - clicker needs time to kill Chrome)
      for (;;) {
        const waitResult = Atomics.wait(this.signal, 0, 0, 5000);

        if (waitResult === 'timed-out') {
          port1.close();
          this.terminate();  // terminate() closes callbackPortMain
          return;
        }

        const sig = Atomics.load(this.signal, 0);

        if (sig === 1) {
          // Quit completed
          port1.close();
          this.callbackPortMain.close();
          this.terminated = true;
          activeBridges.delete(this);
          this.worker.terminate();
          return;
        }

        if (sig === 2) {
          this.handleCallback();
        }
      }
    } catch {
      // If anything fails, force terminate
      this.terminate();
    }
  }

  terminate(): void {
    if (this.terminated) return;
    this.terminated = true;
    this.callbackPortMain.close();
    activeBridges.delete(this);
    this.worker.terminate();
  }
}
