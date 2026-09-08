import { ModelOptions, modelParams } from './model-options';
import { resolve } from 'path';
import { BiDiClient } from './bidi';

export interface CheckOptions extends ModelOptions { /** Read-only archive on the runtime host. */ record?: string; }
export interface RecordedCheckOptions extends CheckOptions { record: string; executablePath?: string; }
export interface CheckResult {
  status: 'passed' | 'failed' | 'inconclusive';
  claim: string;
  summary: string;
  evidence: { type: 'observation'; summary: string }[];
}
export const CHECK_TIMEOUT_MS = 210_000;

/** @internal All inference and browser actions run in the existing Go runtime. */
export function sendCheck(client: BiDiClient, claim: string, options: CheckOptions = {}, context?: string): Promise<CheckResult> {
  const params: Record<string, unknown> = { claim, ...modelParams(options) };
  if (options.record !== undefined) {
    if (!options.record) throw new Error('record must be a nonempty path');
    params.record = resolve(options.record);
  } else if (context) params.context = context;
  return client.send<CheckResult>('vibium:check.run', params, CHECK_TIMEOUT_MS);
}
