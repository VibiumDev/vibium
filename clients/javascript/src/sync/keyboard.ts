import { SyncBridge } from './bridge';

export class KeyboardSync {
  private bridge: SyncBridge;
  private pageId: number;

  constructor(bridge: SyncBridge, pageId: number) {
    this.bridge = bridge;
    this.pageId = pageId;
  }

  press(key: string): void {
    this.bridge.call('keyboard.press', [this.pageId, key]);
  }

  down(key: string): void {
    this.bridge.call('keyboard.down', [this.pageId, key]);
  }

  up(key: string): void {
    this.bridge.call('keyboard.up', [this.pageId, key]);
  }

  type(text: string): void {
    this.bridge.call('keyboard.type', [this.pageId, text]);
  }
}

export class MouseSync {
  private bridge: SyncBridge;
  private pageId: number;

  constructor(bridge: SyncBridge, pageId: number) {
    this.bridge = bridge;
    this.pageId = pageId;
  }

  click(x: number, y: number): void {
    this.bridge.call('mouse.click', [this.pageId, x, y]);
  }

  move(x: number, y: number): void {
    this.bridge.call('mouse.move', [this.pageId, x, y]);
  }

  down(): void {
    this.bridge.call('mouse.down', [this.pageId]);
  }

  up(): void {
    this.bridge.call('mouse.up', [this.pageId]);
  }

  wheel(deltaX: number, deltaY: number): void {
    this.bridge.call('mouse.wheel', [this.pageId, deltaX, deltaY]);
  }
}

export class TouchSync {
  private bridge: SyncBridge;
  private pageId: number;

  constructor(bridge: SyncBridge, pageId: number) {
    this.bridge = bridge;
    this.pageId = pageId;
  }

  tap(x: number, y: number): void {
    this.bridge.call('touch.tap', [this.pageId, x, y]);
  }
}
