import { resolve } from 'path';
import { BiDiClient } from './bidi';

export interface VerifyOptions { /** Read-only archive on the runtime host. */ record?: string; }
export interface RecordedVerifyOptions extends VerifyOptions { record: string; executablePath?: string; }
export interface VerificationResult {
  status: 'passed' | 'failed' | 'inconclusive';
  claim: string;
  summary: string;
  evidence: { type: 'observation'; summary: string }[];
}
export const VERIFY_TIMEOUT_MS = 210_000;

/** @internal All inference and browser actions run in the existing Go runtime. */
export function sendVerification(client: BiDiClient, claim: string, options: VerifyOptions = {}, context?: string): Promise<VerificationResult> {
  const params: Record<string, unknown> = { claim };
  if (options.record !== undefined) {
    if (!options.record) throw new Error('record must be a nonempty path');
    params.record = resolve(options.record);
  } else if (context) params.context = context;
  return client.send<VerificationResult>('vibium:verify.run', params, VERIFY_TIMEOUT_MS);
}
