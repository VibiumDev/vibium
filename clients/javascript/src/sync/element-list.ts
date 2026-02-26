import { SyncBridge } from './bridge';
import { ElementSync } from './element';
import { ElementInfo } from '../element';
import { FilterOptions } from '../element-list';

export class ElementListSync {
  private bridge: SyncBridge;
  private listId: number;

  constructor(bridge: SyncBridge, listId: number) {
    this.bridge = bridge;
    this.listId = listId;
  }

  count(): number {
    const result = this.bridge.call<{ count: number }>('elementList.count', [this.listId]);
    return result.count;
  }

  first(): ElementSync {
    const result = this.bridge.call<{ elementId: number; info: ElementInfo }>('elementList.first', [this.listId]);
    return new ElementSync(this.bridge, result.elementId, result.info);
  }

  last(): ElementSync {
    const result = this.bridge.call<{ elementId: number; info: ElementInfo }>('elementList.last', [this.listId]);
    return new ElementSync(this.bridge, result.elementId, result.info);
  }

  nth(index: number): ElementSync {
    const result = this.bridge.call<{ elementId: number; info: ElementInfo }>('elementList.nth', [this.listId, index]);
    return new ElementSync(this.bridge, result.elementId, result.info);
  }

  filter(opts: FilterOptions): ElementListSync {
    const result = this.bridge.call<{ listId: number; elementIds: number[]; count: number }>('elementList.filter', [this.listId, opts]);
    return new ElementListSync(this.bridge, result.listId);
  }

  toArray(): ElementSync[] {
    const result = this.bridge.call<{ elementIds: number[] }>('elementList.toArray', [this.listId]);
    return result.elementIds.map(id => {
      // We need info for each element — fetch via nth as a workaround
      // Actually toArray returns IDs that are already stored; we create lightweight ElementSync
      return new ElementSync(this.bridge, id, { tag: '', text: '', box: { x: 0, y: 0, width: 0, height: 0 } });
    });
  }

  [Symbol.iterator](): Iterator<ElementSync> {
    const arr = this.toArray();
    let index = 0;
    return {
      next(): IteratorResult<ElementSync> {
        if (index < arr.length) {
          return { value: arr[index++], done: false };
        }
        return { value: undefined as unknown as ElementSync, done: true };
      },
    };
  }
}
