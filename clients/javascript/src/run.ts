import { ModelOptions, modelParams } from './model-options';
import { BiDiClient } from './bidi';

export type RunOptions = ModelOptions;

export interface RunResult {
  status: 'completed' | 'not_completed';
  goal: string;
  summary: string;
  evidence: { type: 'observation'; summary: string }[];
}
export const RUN_TIMEOUT_MS = 210_000;

/** @internal The existing Go runtime owns all provider and browser operations. */
export function sendRun(client: BiDiClient, goal: string, options: RunOptions = {}, context?: string): Promise<RunResult> {
  return client.send<RunResult>('vibium:run.run', { goal, ...modelParams(options), ...(context ? { context } : {}) }, RUN_TIMEOUT_MS);
}
