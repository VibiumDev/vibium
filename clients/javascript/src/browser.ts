import { ClickerProcess } from './clicker';
import { BiDiClient } from './bidi';
import { Vibe } from './vibe';
import { debug, info } from './utils/debug';
import { checkNodeVersion } from './utils/version';

export interface LaunchOptions {
  headless?: boolean;
  port?: number;
  executablePath?: string;
}

export const browser = {
  async launch(options: LaunchOptions = {}): Promise<Vibe> {
    checkNodeVersion();

    const { headless = false, port, executablePath } = options;
    debug('launching browser', { headless, port, executablePath });

    // Start the clicker process
    const process = await ClickerProcess.start({
      headless,
      port,
      executablePath,
    });
    debug('clicker started', { port: process.port });

    // Connect to the proxy
    const client = await BiDiClient.connect(`ws://localhost:${process.port}`);
    info('browser launched', { port: process.port });

    return new Vibe(client, process);
  },
};
