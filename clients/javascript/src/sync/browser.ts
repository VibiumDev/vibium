import { SyncBridge } from './bridge';
import { VibeSync } from './vibe';

export interface LaunchOptions {
  headless?: boolean;
  width?: number;
  height?: number;
}

export const browserSync = {
  launch(options: LaunchOptions = {}): VibeSync {
    const bridge = SyncBridge.create();
    bridge.call('launch', [options]);
    return new VibeSync(bridge);
  },
};
