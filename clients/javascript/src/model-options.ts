/** Per-call settings. Omitted values use this operation's runtime environment. */
export interface ModelOptions {
  provider?: 'openai' | 'xai' | 'anthropic' | 'google' | 'openai-compatible' | 'local';
  model?: string;
  /** The AI provider's API base URL (the model endpoint, not the site under
   * test). Empty string resets the endpoint to the provider default. */
  aiBaseURL?: string;
  /** Empty string clears an inherited effort. */
  reasoningEffort?: string;
}

/** @internal Only public model settings cross the existing runtime transport. */
export function modelParams(options: ModelOptions): Record<string, unknown> {
  return Object.fromEntries(['provider', 'model', 'aiBaseURL', 'reasoningEffort']
    .filter(key => options[key as keyof ModelOptions] !== undefined)
    .map(key => [key, options[key as keyof ModelOptions]]));
}
