import { parentPort, workerData } from 'worker_threads';
import { browser } from '../browser';
import { Vibe, FindOptions, RecordingOptions } from '../vibe';
import { Element, ActionOptions } from '../element';

interface WorkerData {
  signal: Int32Array;
}

interface Command {
  id: number;
  method: string;
  args: unknown[];
}

const { signal } = workerData as WorkerData;

let vibe: Vibe | null = null;
let elements: Map<number, Element> = new Map();
let nextElementId = 1;

async function handleCommand(cmd: Command): Promise<unknown> {
  switch (cmd.method) {
    case 'launch': {
      const [options] = cmd.args as [{ headless?: boolean } | undefined];
      vibe = await browser.launch(options);
      return { success: true };
    }

    case 'go': {
      if (!vibe) throw new Error('Browser not launched');
      const [url] = cmd.args as [string];
      await vibe.go(url);
      return { success: true };
    }

    case 'screenshot': {
      if (!vibe) throw new Error('Browser not launched');
      const buffer = await vibe.screenshot();
      return { data: buffer.toString('base64') };
    }

    case 'evaluate': {
      if (!vibe) throw new Error('Browser not launched');
      const [script] = cmd.args as [string];
      const result = await vibe.evaluate(script);
      return { result };
    }

    case 'find': {
      if (!vibe) throw new Error('Browser not launched');
      const [selector, options] = cmd.args as [string, FindOptions | undefined];
      const element = await vibe.find(selector, options);
      const elementId = nextElementId++;
      elements.set(elementId, element);
      return { elementId, info: element.info };
    }

    case 'element.click': {
      const [elementId, options] = cmd.args as [number, ActionOptions | undefined];
      const element = elements.get(elementId);
      if (!element) throw new Error(`Element ${elementId} not found`);
      await element.click(options);
      return { success: true };
    }

    case 'element.type': {
      const [elementId, text, options] = cmd.args as [number, string, ActionOptions | undefined];
      const element = elements.get(elementId);
      if (!element) throw new Error(`Element ${elementId} not found`);
      await element.type(text, options);
      return { success: true };
    }

    case 'element.text': {
      const [elementId] = cmd.args as [number];
      const element = elements.get(elementId);
      if (!element) throw new Error(`Element ${elementId} not found`);
      const text = await element.text();
      return { text };
    }

    case 'element.getAttribute': {
      const [elementId, name] = cmd.args as [number, string];
      const element = elements.get(elementId);
      if (!element) throw new Error(`Element ${elementId} not found`);
      const value = await element.getAttribute(name);
      return { value };
    }

    case 'element.boundingBox': {
      const [elementId] = cmd.args as [number];
      const element = elements.get(elementId);
      if (!element) throw new Error(`Element ${elementId} not found`);
      const box = await element.boundingBox();
      return { box };
    }

    case 'startRecording': {
      if (!vibe) throw new Error('Browser not launched');
      const [options] = cmd.args as [RecordingOptions | undefined];
      await vibe.startRecording(options);
      return { success: true };
    }

    case 'stopRecording': {
      if (!vibe) throw new Error('Browser not launched');
      const outputPath = await vibe.stopRecording();
      return { outputPath };
    }

    case 'quit': {
      if (!vibe) throw new Error('Browser not launched');
      await vibe.quit();
      vibe = null;
      elements.clear();
      return { success: true };
    }

    default:
      throw new Error(`Unknown method: ${cmd.method}`);
  }
}

parentPort!.on('message', async ({ cmd, port }: { cmd: Command; port: MessagePort }) => {
  let result: unknown;
  let error: string | null = null;

  try {
    result = await handleCommand(cmd);
  } catch (err) {
    error = err instanceof Error ? err.message : String(err);
  }

  // Send result through the port
  port.postMessage({ result, error });

  // Signal completion
  Atomics.store(signal, 0, 1);
  Atomics.notify(signal, 0);
});
