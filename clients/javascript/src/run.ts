import { ModelOptions, modelParams } from './model-options';
import { BiDiClient } from './bidi';

export interface RunOptions extends ModelOptions {
  /** Site under test: opened first unless the current page already shares its
   * origin; relative navigation paths resolve against it. Not the AI provider
   * endpoint; that is aiBaseURL. */
  baseURL?: string;
}

export interface RunResult {
  status: 'completed' | 'not_completed';
  goal: string;
  summary: string;
  evidence: { type: 'observation'; summary: string }[];
}
export const RUN_TIMEOUT_MS = 210_000;

/** @internal The existing Go runtime owns all provider and browser operations. */
export function sendRun(client: BiDiClient, goal: string, options: RunOptions = {}, context?: string): Promise<RunResult> {
  const params: Record<string, unknown> = { goal, ...modelParams(options), ...(context ? { context } : {}) };
  if (options.baseURL !== undefined) params.baseURL = options.baseURL;
  return client.send<RunResult>('vibium:run.run', params, RUN_TIMEOUT_MS);
}
