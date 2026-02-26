type MessageHandler = (data: string, info: { direction: 'sent' | 'received' }) => void;
type CloseHandler = (code?: number, reason?: string) => void;

export class WebSocketInfo {
  private _url: string;
  private _isClosed = false;
  private messageHandlers: MessageHandler[] = [];
  private closeHandlers: CloseHandler[] = [];

  constructor(url: string) {
    this._url = url;
  }

  url(): string {
    return this._url;
  }

  onMessage(fn: MessageHandler): void {
    this.messageHandlers.push(fn);
  }

  onClose(fn: CloseHandler): void {
    this.closeHandlers.push(fn);
  }

  isClosed(): boolean {
    return this._isClosed;
  }

  /** @internal */
  _emitMessage(data: string, direction: 'sent' | 'received'): void {
    for (const fn of this.messageHandlers) fn(data, { direction });
  }

  /** @internal */
  _emitClose(code?: number, reason?: string): void {
    this._isClosed = true;
    for (const fn of this.closeHandlers) fn(code, reason);
  }
}
