import { BiDiClient } from './bidi';
import * as fs from 'fs/promises';

/** Represents a file download triggered by the page. */
export class Download {
  private client: BiDiClient;
  private _url: string;
  private _suggestedFilename: string;
  private _resolve!: (value: { status: string; filepath: string | null }) => void;
  private _completionPromise: Promise<{ status: string; filepath: string | null }>;

  constructor(client: BiDiClient, url: string, suggestedFilename: string) {
    this.client = client;
    this._url = url;
    this._suggestedFilename = suggestedFilename;
    this._completionPromise = new Promise((resolve) => {
      this._resolve = resolve;
    });
  }

  /** The URL of the download. */
  url(): string {
    return this._url;
  }

  /** The filename suggested by the server (from Content-Disposition). */
  suggestedFilename(): string {
    return this._suggestedFilename;
  }

  /** Wait for the download to complete, then save to the specified path. */
  async saveAs(path: string): Promise<void> {
    const result = await this._completionPromise;
    if (result.status !== 'complete' || !result.filepath) {
      throw new Error(`Download failed with status: ${result.status}`);
    }

    await this.client.send('vibium:download.saveAs', {
      sourcePath: result.filepath,
      destPath: path,
    });
  }

  /** Wait for the download to complete and return the temp file path, or null if failed. */
  async path(): Promise<string | null> {
    const result = await this._completionPromise;
    return result.filepath;
  }

  /** Internal: called by Page when downloadCompleted fires. */
  _complete(status: string, filepath: string | null): void {
    this._resolve({ status, filepath });
  }
}
