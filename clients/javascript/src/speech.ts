import { resolve } from 'path';
import { BiDiClient } from './bidi';
import { VibiumProcess } from './clicker';
import { BiDiError, SpeechError } from './utils/errors';

export interface SpeechOptions {
  /** 'session', 'local', or a path ending in .wav. */
  target?: string;
  provider?: string;
  model?: string;
  language?: string;
}

export interface SayOptions extends SpeechOptions {
  /** A WAV file to play as it is. Pass an empty text with it. */
  file?: string;
  voice?: string;
  speed?: number;
  /** false renders again and replaces the cached audio. Default true. */
  cache?: boolean;
  cacheDir?: string;
}

export interface SayResult {
  text: string;
  target: string;
  provider: string;
  model: string;
  voice: string;
  durationMs: number;
  cached: boolean;
  /** SHA-256 of the audio that was played. */
  sha256: string;
  /** Present for file targets. */
  path?: string;
  /** Present when a file was played in place of text. */
  file?: string;
}

export interface ListenOptions extends SpeechOptions {
  /** Trailing silence that ends the reply, in ms. 0 turns endpointing off. Default 1000. */
  silence?: number;
  /** How long to wait for speech to start, from the call, in ms. Default 10000. */
  timeout?: number;
  /** Maximum capture length from the call, in ms. Default 30000. */
  duration?: number;
  /** Return an empty transcript instead of throwing when nothing is heard. */
  allowSilence?: boolean;
  /** Also save the captured audio as a WAV file. */
  path?: string;
  /** false captures without a transcript. Default true. */
  transcribe?: boolean;
}

export interface ListenResult {
  text: string;
  /** False when the capture was not sent to a provider. */
  transcribed: boolean;
  /** False when no speech was detected (only with allowSilence). */
  speech: boolean;
  /** Times are ms from the start of the capture. */
  segments: { startMs: number; endMs: number; text: string }[];
  /** Present when the provider returns word timings. */
  words?: { startMs: number; endMs: number; word: string }[];
  /** When speech started and ended in the capture. */
  onsetMs?: number;
  speechEndMs?: number;
  /** Speech onset minus the end of the preceding say. */
  latencyMs?: number;
  stoppedBy: 'silence' | 'duration' | 'eof' | 'timeout';
  /** Length of the captured audio. */
  durationMs: number;
  target: string;
  provider: string;
  model: string;
  path?: string;
  /** Audio the target dropped on its way here, in ms. The transcript may miss words. */
  lostMs?: number;
}

export const SAY_TIMEOUT_MS = 180_000;
const LISTEN_TIMEOUT_MS = 10_000;
const LISTEN_DURATION_MS = 30_000;

/** @internal A listen may wait its whole timeout, capture its whole duration, and then transcribe. */
export function listenTimeoutMs(options: ListenOptions = {}): number {
  return (options.timeout ?? LISTEN_TIMEOUT_MS) + (options.duration ?? LISTEN_DURATION_MS) + 60_000;
}

/** @internal The sync bridge outwaits the command, so the command's own timeout is the error seen. */
export const BRIDGE_SLACK_MS = 5_000;

const isWAVPath = (target: string) => /\.wav$/i.test(target);

/** Options as wire params. The engine may run in another directory, so paths go absolute. */
function speechParams(options: object, paths: string[], context?: string): Record<string, unknown> {
  const params: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(options)) {
    if (value === undefined) continue;
    const isPath = paths.includes(key) || (key === 'target' && typeof value === 'string' && isWAVPath(value));
    params[key] = isPath && typeof value === 'string' && value !== '' ? resolve(value) : value;
  }
  if (context) params.context = context;
  return params;
}

async function speechCall<T>(client: BiDiClient, method: string, params: Record<string, unknown>, timeout: number): Promise<T> {
  try {
    return await client.send<T>(method, params, timeout);
  } catch (err) {
    // The engine answers a speech failure with its code in place of the
    // generic 'error'.
    if (err instanceof BiDiError && err.error !== 'error') {
      throw new SpeechError(err.error, err.message.slice(err.error.length + 2));
    }
    throw err;
  }
}

/** @internal Rendering, playback and capture all run in the Go runtime. */
export function sendSay(client: BiDiClient, text: string, options: SayOptions = {}, context?: string): Promise<SayResult> {
  const params = speechParams(options, ['file', 'cacheDir'], context);
  if (text) params.text = text;
  return speechCall<SayResult>(client, 'vibium:speech.say', params, SAY_TIMEOUT_MS);
}

/** @internal */
export function sendListen(client: BiDiClient, options: ListenOptions = {}, context?: string): Promise<ListenResult> {
  return speechCall<ListenResult>(client, 'vibium:speech.listen', speechParams(options, ['path'], context), listenTimeoutMs(options));
}

interface Engine { proc: VibiumProcess; client: BiDiClient; }

// One engine with no browser serves every module-level say and listen, so
// the capture window a say opens is still there for the next listen.
let engine: Promise<Engine> | null = null;

async function startEngine(): Promise<Engine> {
  const proc = await VibiumProcess.start({ noBrowser: true });
  proc.unref();
  return { proc, client: BiDiClient.fromStreams(proc.stdin, proc.stdout, proc.preReadyLines) };
}

/** @internal The shared engine, started on first use and again if it has exited. */
export async function speechEngine(): Promise<BiDiClient> {
  const seen = engine;
  const running = seen && await seen.catch(() => null);
  if (running && !running.proc.exited) return running.client;
  if (engine === seen) engine = startEngine();
  return (await engine!).client;
}

/** @internal */
export async function stopSpeechEngine(): Promise<void> {
  const seen = engine;
  engine = null;
  const running = seen && await seen.catch(() => null);
  if (!running) return;
  try { await running.client.close(); } finally { await running.proc.stop(); }
}
