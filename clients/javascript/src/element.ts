import { BiDiClient } from './bidi';
import { ElementList, ElementListInfo } from './element-list';

export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface ElementInfo {
  tag: string;
  text: string;
  box: BoundingBox;
}

export interface ActionOptions {
  /** Timeout in milliseconds for actionability checks. Default: 30000 */
  timeout?: number;
}

export interface SelectorOptions {
  role?: string;
  text?: string;
  label?: string;
  placeholder?: string;
  alt?: string;
  title?: string;
  testid?: string;
  xpath?: string;
  near?: string;
  timeout?: number;
}

export class Element {
  private client: BiDiClient;
  private context: string;
  private selector: string;
  private _index?: number;
  private _params: Record<string, unknown>;
  readonly info: ElementInfo;

  constructor(
    client: BiDiClient,
    context: string,
    selector: string,
    info: ElementInfo,
    index?: number,
    params?: Record<string, unknown>
  ) {
    this.client = client;
    this.context = context;
    this.selector = selector;
    this.info = info;
    this._index = index;
    this._params = params || {};
  }

  /** Build the common params sent to vibium: commands for element resolution. */
  private commandParams(extra?: Record<string, unknown>): Record<string, unknown> {
    return {
      ...this._params,
      context: this.context,
      selector: this.selector,
      index: this._index,
      ...extra,
    };
  }

  /** Return params that can identify this element for use as a target (e.g. dragTo). */
  toParams(): Record<string, unknown> {
    return {
      ...this._params,
      selector: this.selector,
      index: this._index,
    };
  }

  /**
   * Click the element.
   * Waits for element to be visible, stable, receive events, and enabled.
   */
  async click(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:click', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Double-click the element. */
  async dblclick(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:dblclick', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /**
   * Fill the element with text (clears existing content first).
   * For inputs and textareas.
   */
  async fill(value: string, options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:fill', this.commandParams({
      value,
      timeout: options?.timeout,
    }));
  }

  /**
   * Type text into the element (appends to existing content).
   * Waits for element to be visible, stable, receive events, enabled, and editable.
   */
  async type(text: string, options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:type', this.commandParams({
      text,
      timeout: options?.timeout,
    }));
  }

  /**
   * Press a key while the element is focused.
   * Supports key names ("Enter", "Tab") and combos ("Control+a").
   */
  async press(key: string, options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:press', this.commandParams({
      key,
      timeout: options?.timeout,
    }));
  }

  /** Clear the element's content (select all + delete). */
  async clear(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:clear', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Check a checkbox (no-op if already checked). */
  async check(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:check', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Uncheck a checkbox (no-op if already unchecked). */
  async uncheck(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:uncheck', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Select an option in a <select> element by value. */
  async selectOption(value: string, options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:selectOption', this.commandParams({
      value,
      timeout: options?.timeout,
    }));
  }

  /** Hover over the element (move mouse to center, no click). */
  async hover(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:hover', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Focus the element. */
  async focus(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:focus', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Drag this element to a target element. */
  async dragTo(target: Element, options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:dragTo', this.commandParams({
      target: target.toParams(),
      timeout: options?.timeout,
    }));
  }

  /** Tap the element (touch action). */
  async tap(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:tap', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Scroll the element into view. */
  async scrollIntoView(options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:scrollIntoView', this.commandParams({
      timeout: options?.timeout,
    }));
  }

  /** Dispatch a DOM event on the element. */
  async dispatchEvent(eventType: string, eventInit?: Record<string, unknown>, options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:dispatchEvent', this.commandParams({
      eventType,
      eventInit,
      timeout: options?.timeout,
    }));
  }

  /** Set files on an <input type="file"> element. */
  async setFiles(files: string[], options?: ActionOptions): Promise<void> {
    await this.client.send('vibium:el.setFiles', this.commandParams({
      files,
      timeout: options?.timeout,
    }));
  }

  // --- Element State ---

  /** Get the element's textContent (trimmed). */
  async text(): Promise<string> {
    const result = await this.client.send<{ text: string }>('vibium:el.text', this.commandParams());
    return result.text;
  }

  /** Get the element's innerText (rendered text only). */
  async innerText(): Promise<string> {
    const result = await this.client.send<{ text: string }>('vibium:el.innerText', this.commandParams());
    return result.text;
  }

  /** Get the element's innerHTML. */
  async html(): Promise<string> {
    const result = await this.client.send<{ html: string }>('vibium:el.html', this.commandParams());
    return result.html;
  }

  /** Get the element's value (for inputs, textareas, selects). */
  async value(): Promise<string> {
    const result = await this.client.send<{ value: string }>('vibium:el.value', this.commandParams());
    return result.value;
  }

  /** Get an attribute value. Short name for getAttribute. */
  async attr(name: string): Promise<string | null> {
    const result = await this.client.send<{ value: string | null }>('vibium:el.attr', this.commandParams({ name }));
    return result.value;
  }

  /** Get an attribute value. Alias for attr(). */
  async getAttribute(name: string): Promise<string | null> {
    return this.attr(name);
  }

  /** Get the element's bounding box. Short name for boundingBox. */
  async bounds(): Promise<BoundingBox> {
    const result = await this.client.send<BoundingBox>('vibium:el.bounds', this.commandParams());
    return result;
  }

  /** Get the element's bounding box. Alias for bounds(). */
  async boundingBox(): Promise<BoundingBox> {
    return this.bounds();
  }

  /** Check if the element is visible (has dimensions, not display:none/visibility:hidden/opacity:0). */
  async isVisible(): Promise<boolean> {
    const result = await this.client.send<{ visible: boolean }>('vibium:el.isVisible', this.commandParams());
    return result.visible;
  }

  /** Check if the element is hidden (inverse of isVisible). */
  async isHidden(): Promise<boolean> {
    const result = await this.client.send<{ hidden: boolean }>('vibium:el.isHidden', this.commandParams());
    return result.hidden;
  }

  /** Check if the element is enabled (not disabled). */
  async isEnabled(): Promise<boolean> {
    const result = await this.client.send<{ enabled: boolean }>('vibium:el.isEnabled', this.commandParams());
    return result.enabled;
  }

  /** Check if the element is checked (for checkboxes/radios). */
  async isChecked(): Promise<boolean> {
    const result = await this.client.send<{ checked: boolean }>('vibium:el.isChecked', this.commandParams());
    return result.checked;
  }

  /** Check if the element is editable (not disabled and not readonly). */
  async isEditable(): Promise<boolean> {
    const result = await this.client.send<{ editable: boolean }>('vibium:el.isEditable', this.commandParams());
    return result.editable;
  }

  /** Get the element's computed ARIA role. */
  async role(): Promise<string> {
    const result = await this.client.send<{ role: string }>('vibium:el.role', this.commandParams());
    return result.role;
  }

  /** Get the element's accessible name (label). */
  async label(): Promise<string> {
    const result = await this.client.send<{ label: string }>('vibium:el.label', this.commandParams());
    return result.label;
  }

  /** Evaluate a function with this element as argument. The fn body receives `el`. */
  async eval<T = unknown>(fn: string): Promise<T> {
    const result = await this.client.send<{ value: T }>('vibium:el.eval', this.commandParams({ fn }));
    return result.value;
  }

  /** Take a screenshot of just this element. Returns a PNG buffer. */
  async screenshot(): Promise<Buffer> {
    const result = await this.client.send<{ data: string }>('vibium:el.screenshot', this.commandParams());
    return Buffer.from(result.data, 'base64');
  }

  /** Wait until the element reaches a state: "visible", "hidden", "attached", or "detached". */
  async waitUntil(state?: string, options?: { timeout?: number }): Promise<void> {
    await this.client.send('vibium:el.waitFor', this.commandParams({
      state,
      timeout: options?.timeout,
    }));
  }

  /** Find a child element by CSS selector or semantic options. Scoped to this element. */
  find(selector: string | SelectorOptions, options?: { timeout?: number }): FluentElement {
    const promise = (async () => {
      const params: Record<string, unknown> = {
        context: this.context,
        scope: this.selector,
        timeout: options?.timeout,
      };

      if (typeof selector === 'string') {
        params.selector = selector;
      } else {
        Object.assign(params, selector);
        if (selector.timeout) params.timeout = selector.timeout;
      }

      const result = await this.client.send<{
        tag: string;
        text: string;
        box: BoundingBox;
      }>('vibium:find', params);

      const info: ElementInfo = { tag: result.tag, text: result.text, box: result.box };
      const childSelector = typeof selector === 'string' ? selector : '';
      const childParams = typeof selector === 'string' ? { selector } : { ...selector };
      return new Element(this.client, this.context, childSelector, info, undefined, childParams);
    })();
    return fluent(promise);
  }

  /** Find all child elements by CSS selector or semantic options. Scoped to this element. */
  async findAll(selector: string | SelectorOptions, options?: { timeout?: number }): Promise<ElementList> {
    const params: Record<string, unknown> = {
      context: this.context,
      scope: this.selector,
      timeout: options?.timeout,
    };

    if (typeof selector === 'string') {
      params.selector = selector;
    } else {
      Object.assign(params, selector);
      if (selector.timeout) params.timeout = selector.timeout;
    }

    const result = await this.client.send<{
      elements: Array<{ tag: string; text: string; box: BoundingBox; index: number }>;
      count: number;
    }>('vibium:findAll', params);

    const selectorStr = typeof selector === 'string' ? selector : '';
    const selectorParams = typeof selector === 'string' ? { selector } : { ...selector };
    const elements = result.elements.map((el) => {
      const info: ElementInfo = { tag: el.tag, text: el.text, box: el.box };
      return new Element(this.client, this.context, selectorStr, info, el.index, selectorParams);
    });

    return new ElementList(this.client, this.context, selector, elements);
  }

  private getCenter(): { x: number; y: number } {
    return {
      x: this.info.box.x + this.info.box.width / 2,
      y: this.info.box.y + this.info.box.height / 2,
    };
  }
}

/** A Promise<Element> that also exposes Element methods for chaining. */
export type FluentElement = Promise<Element> & {
  // Interaction
  click(options?: ActionOptions): Promise<void>;
  dblclick(options?: ActionOptions): Promise<void>;
  fill(value: string, options?: ActionOptions): Promise<void>;
  type(text: string, options?: ActionOptions): Promise<void>;
  press(key: string, options?: ActionOptions): Promise<void>;
  clear(options?: ActionOptions): Promise<void>;
  check(options?: ActionOptions): Promise<void>;
  uncheck(options?: ActionOptions): Promise<void>;
  selectOption(value: string, options?: ActionOptions): Promise<void>;
  hover(options?: ActionOptions): Promise<void>;
  focus(options?: ActionOptions): Promise<void>;
  dragTo(target: Element, options?: ActionOptions): Promise<void>;
  tap(options?: ActionOptions): Promise<void>;
  scrollIntoView(options?: ActionOptions): Promise<void>;
  dispatchEvent(eventType: string, eventInit?: Record<string, unknown>, options?: ActionOptions): Promise<void>;
  setFiles(files: string[], options?: ActionOptions): Promise<void>;
  // State
  text(): Promise<string>;
  innerText(): Promise<string>;
  html(): Promise<string>;
  value(): Promise<string>;
  attr(name: string): Promise<string | null>;
  getAttribute(name: string): Promise<string | null>;
  bounds(): Promise<BoundingBox>;
  boundingBox(): Promise<BoundingBox>;
  isVisible(): Promise<boolean>;
  isHidden(): Promise<boolean>;
  isEnabled(): Promise<boolean>;
  isChecked(): Promise<boolean>;
  isEditable(): Promise<boolean>;
  role(): Promise<string>;
  label(): Promise<string>;
  eval<T = unknown>(fn: string): Promise<T>;
  screenshot(): Promise<Buffer>;
  waitUntil(state?: string, options?: { timeout?: number }): Promise<void>;
  // Finding
  find(selector: string | SelectorOptions, options?: { timeout?: number }): FluentElement;
  findAll(selector: string | SelectorOptions, options?: { timeout?: number }): Promise<ElementList>;
};

export function fluent(promise: Promise<Element>): FluentElement {
  const p = promise as FluentElement;
  // Interaction
  p.click = (opts?) => promise.then(el => el.click(opts));
  p.dblclick = (opts?) => promise.then(el => el.dblclick(opts));
  p.fill = (value, opts?) => promise.then(el => el.fill(value, opts));
  p.type = (text, opts?) => promise.then(el => el.type(text, opts));
  p.press = (key, opts?) => promise.then(el => el.press(key, opts));
  p.clear = (opts?) => promise.then(el => el.clear(opts));
  p.check = (opts?) => promise.then(el => el.check(opts));
  p.uncheck = (opts?) => promise.then(el => el.uncheck(opts));
  p.selectOption = (value, opts?) => promise.then(el => el.selectOption(value, opts));
  p.hover = (opts?) => promise.then(el => el.hover(opts));
  p.focus = (opts?) => promise.then(el => el.focus(opts));
  p.dragTo = (target, opts?) => promise.then(el => el.dragTo(target, opts));
  p.tap = (opts?) => promise.then(el => el.tap(opts));
  p.scrollIntoView = (opts?) => promise.then(el => el.scrollIntoView(opts));
  p.dispatchEvent = (type, init?, opts?) => promise.then(el => el.dispatchEvent(type, init, opts));
  p.setFiles = (files, opts?) => promise.then(el => el.setFiles(files, opts));
  // State
  p.text = () => promise.then(el => el.text());
  p.innerText = () => promise.then(el => el.innerText());
  p.html = () => promise.then(el => el.html());
  p.value = () => promise.then(el => el.value());
  p.attr = (name) => promise.then(el => el.attr(name));
  p.getAttribute = (name) => promise.then(el => el.getAttribute(name));
  p.bounds = () => promise.then(el => el.bounds());
  p.boundingBox = () => promise.then(el => el.boundingBox());
  p.isVisible = () => promise.then(el => el.isVisible());
  p.isHidden = () => promise.then(el => el.isHidden());
  p.isEnabled = () => promise.then(el => el.isEnabled());
  p.isChecked = () => promise.then(el => el.isChecked());
  p.isEditable = () => promise.then(el => el.isEditable());
  p.role = () => promise.then(el => el.role());
  p.label = () => promise.then(el => el.label());
  p.eval = (fn) => promise.then(el => el.eval(fn));
  p.screenshot = () => promise.then(el => el.screenshot());
  p.waitUntil = (state?, opts?) => promise.then(el => el.waitUntil(state, opts));
  // Finding
  p.find = (sel, opts?) => fluent(promise.then(el => el.find(sel, opts)));
  p.findAll = (sel, opts?) => promise.then(el => el.findAll(sel, opts));
  return p;
}

export { ElementList } from './element-list';
