import { BiDiClient } from './bidi';

/** Default timeout for download completion (5 minutes). */
const DOWNLOAD_TIMEOUT_MS = 300_000;

/** Represents a file download triggered by the page. */
export class Download {
  private client: BiDiClient;
  private _url: string;
  private _suggestedFilename: string;
  private _navigation: string;

  constructor(client: BiDiClient, params: Record<string, unknown>) {
    this.client = client;
    this._url = (params.url as string) ?? '';
    this._suggestedFilename = (params.suggestedFilename as string) ?? '';
    this._navigation = (params.navigation as string) ?? '';
  }

  /** The URL of the download. */
  url(): string {
    return this._url;
  }

  /** The filename suggested by the server (from Content-Disposition). */
  suggestedFilename(): string {
    return this._suggestedFilename;
  }

  /**
   * Completion is awaited in the engine (vibium:download.await), which
   * tracks every download by its navigation id, so no client keeps a
   * pending-downloads map. The engine answers finished downloads
   * immediately, so asking repeatedly is fine.
   */
  private _waitForCompletion(): Promise<{ status: string; filepath: string | null }> {
    return this.client.send<{ status: string; filepath?: string }>(
      'vibium:download.await',
      { navigation: this._navigation, timeout: DOWNLOAD_TIMEOUT_MS },
      DOWNLOAD_TIMEOUT_MS + 10_000,
    ).then((result) => ({ status: result.status, filepath: result.filepath || null }));
  }

  /** Wait for the download to complete, then save to the specified path. */
  async saveAs(path: string): Promise<void> {
    const result = await this._waitForCompletion();
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
    const result = await this._waitForCompletion();
    return result.filepath;
  }
}
