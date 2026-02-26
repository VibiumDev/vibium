/** Represents a single console message from the page (console.log, warn, error, etc.). */
export class ConsoleMessage {
  private data: Record<string, unknown>;

  constructor(data: Record<string, unknown>) {
    this.data = data;
  }

  /** The console method: 'log', 'warn', 'error', 'debug', 'info', etc. */
  type(): string {
    return (this.data.method as string) ?? 'log';
  }

  /** The message text. */
  text(): string {
    return (this.data.text as string) ?? '';
  }

  /** The serialized arguments passed to the console call. */
  args(): unknown[] {
    return (this.data.args as unknown[]) ?? [];
  }

  /** The source location of the console call, if available. */
  location(): { url: string; lineNumber: number; columnNumber: number } | null {
    const stack = this.data.stackTrace as { callFrames?: { url: string; lineNumber: number; columnNumber: number }[] } | undefined;
    const frame = stack?.callFrames?.[0];
    if (!frame) return null;
    return { url: frame.url, lineNumber: frame.lineNumber, columnNumber: frame.columnNumber };
  }
}
