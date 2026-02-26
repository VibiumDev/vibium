import { BiDiClient } from './bidi';

/** Check if an error is a benign race condition that should be silently ignored. */
function isDialogRaceError(e: unknown): boolean {
  if (!(e instanceof Error)) return false;
  const msg = e.message;
  return msg === 'Connection closed' ||
    msg.includes('no such alert') ||
    msg.includes('No dialog');
}

/** Represents a browser dialog (alert, confirm, prompt, beforeunload). */
export class Dialog {
  private client: BiDiClient;
  private contextId: string;
  private data: Record<string, unknown>;

  constructor(client: BiDiClient, contextId: string, data: Record<string, unknown>) {
    this.client = client;
    this.contextId = contextId;
    this.data = data;
  }

  /** The dialog message text. */
  message(): string {
    return (this.data.message as string) ?? '';
  }

  /** The dialog type: 'alert', 'confirm', 'prompt', or 'beforeunload'. */
  type(): string {
    return (this.data.type as string) ?? 'alert';
  }

  /** The default value for prompt dialogs. */
  defaultValue(): string {
    return (this.data.defaultValue as string) ?? '';
  }

  /** Accept the dialog. For prompt dialogs, optionally provide text. */
  async accept(promptText?: string): Promise<void> {
    try {
      const params: Record<string, unknown> = {
        context: this.contextId,
        accept: true,
      };
      if (promptText !== undefined) params.userText = promptText;
      await this.client.send('browsingContext.handleUserPrompt', params);
    } catch (e) {
      if (isDialogRaceError(e)) return;
      throw e;
    }
  }

  /** Dismiss the dialog (cancel/close). */
  async dismiss(): Promise<void> {
    try {
      await this.client.send('browsingContext.handleUserPrompt', {
        context: this.contextId,
        accept: false,
      });
    } catch (e) {
      if (isDialogRaceError(e)) return;
      throw e;
    }
  }
}
