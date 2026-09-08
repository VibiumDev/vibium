/** Per-call settings. Omitted values use this operation's runtime environment. */
export interface ModelOptions {
  provider?: 'openai' | 'anthropic' | 'google' | 'openai-compatible' | 'local';
  model?: string;
  /** Empty string resets the endpoint to the provider default. */
  baseURL?: string;
  /** Empty string clears an inherited effort. */
  reasoningEffort?: string;
}

/** @internal Only public model settings cross the existing runtime transport. */
export function modelParams(options: ModelOptions): Record<string, unknown> {
  return Object.fromEntries(['provider', 'model', 'baseURL', 'reasoningEffort']
    .filter(key => options[key as keyof ModelOptions] !== undefined)
    .map(key => [key, options[key as keyof ModelOptions]]));
}
