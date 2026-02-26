import { BiDiClient } from './bidi';
import { Element, ElementInfo, SelectorOptions } from './element';

export interface ElementListInfo {
  tag: string;
  text: string;
  box: { x: number; y: number; width: number; height: number };
  index: number;
}

export interface FilterOptions {
  hasText?: string;
  has?: string;
}

export class ElementList {
  private client: BiDiClient;
  private context: string;
  private selector: string | SelectorOptions;
  private elements: Element[];

  constructor(
    client: BiDiClient,
    context: string,
    selector: string | SelectorOptions,
    elements: Element[]
  ) {
    this.client = client;
    this.context = context;
    this.selector = selector;
    this.elements = elements;
  }

  /** Number of elements in the list. */
  count(): number {
    return this.elements.length;
  }

  /** First element in the list. */
  first(): Element {
    return this.nth(0);
  }

  /** Last element in the list. */
  last(): Element {
    return this.nth(this.elements.length - 1);
  }

  /** Element at the given index. */
  nth(index: number): Element {
    if (index < 0 || index >= this.elements.length) {
      throw new Error(`Index ${index} out of bounds (0..${this.elements.length - 1})`);
    }
    return this.elements[index];
  }

  /** Filter elements by re-querying with filter params. */
  async filter(opts: FilterOptions): Promise<ElementList> {
    const params: Record<string, unknown> = {
      context: this.context,
      timeout: 5000,
    };

    // Merge original selector params
    if (typeof this.selector === 'string') {
      params.selector = this.selector;
    } else {
      Object.assign(params, this.selector);
    }

    if (opts.hasText) params.hasText = opts.hasText;
    if (opts.has) params.has = opts.has;

    const result = await this.client.send<{
      elements: ElementListInfo[];
      count: number;
    }>('vibium:findAll', params);

    const elements = result.elements.map((el) => {
      const info: ElementInfo = { tag: el.tag, text: el.text, box: el.box };
      const selectorStr = typeof this.selector === 'string' ? this.selector : '';
      const selectorParams = typeof this.selector === 'string' ? { selector: this.selector } : { ...this.selector };
      return new Element(this.client, this.context, selectorStr, info, el.index, selectorParams);
    });

    return new ElementList(this.client, this.context, this.selector, elements);
  }

  /** Return elements as an array. */
  toArray(): Element[] {
    return [...this.elements];
  }

  [Symbol.iterator](): Iterator<Element> {
    let index = 0;
    const elements = this.elements;
    return {
      next(): IteratorResult<Element> {
        if (index < elements.length) {
          return { value: elements[index++], done: false };
        }
        return { value: undefined as unknown as Element, done: true };
      },
    };
  }
}
