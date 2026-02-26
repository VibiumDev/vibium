/** Serialized dialog data sent from worker to main thread. */
export interface DialogData {
  type: string;
  message: string;
  defaultValue: string;
}

/** Decision returned by a sync dialog handler. */
export interface DialogDecision {
  action: 'accept' | 'dismiss';
  promptText?: string;
}

/**
 * Sync wrapper for a browser dialog (alert, confirm, prompt, beforeunload).
 * The user's handler calls accept() or dismiss() to set the decision.
 * If none is called, the default is 'dismiss'.
 */
export class DialogSync {
  private data: DialogData;
  /** @internal */
  _decision: DialogDecision = { action: 'dismiss' };

  constructor(data: DialogData) {
    this.data = data;
  }

  /** The dialog type: 'alert', 'confirm', 'prompt', or 'beforeunload'. */
  type(): string {
    return this.data.type;
  }

  /** The dialog message text. */
  message(): string {
    return this.data.message;
  }

  /** The default value for prompt dialogs. */
  defaultValue(): string {
    return this.data.defaultValue;
  }

  /** Accept the dialog. For prompt dialogs, optionally provide text. */
  accept(promptText?: string): void {
    this._decision = { action: 'accept' };
    if (promptText !== undefined) this._decision.promptText = promptText;
  }

  /** Dismiss the dialog (cancel/close). */
  dismiss(): void {
    this._decision = { action: 'dismiss' };
  }
}
