import { SyncBridge } from './bridge';
import { ElementListSync } from './element-list';
import { ActionOptions, BoundingBox, ElementInfo, SelectorOptions } from '../element';

export class ElementSync {
  private bridge: SyncBridge;
  private elementId: number;
  readonly info: ElementInfo;

  constructor(bridge: SyncBridge, elementId: number, info: ElementInfo) {
    this.bridge = bridge;
    this.elementId = elementId;
    this.info = info;
  }

  /**
   * Click the element.
   * Waits for element to be visible, stable, receive events, and enabled.
   */
  click(options?: ActionOptions): void {
    this.bridge.call('element.click', [this.elementId, options]);
  }

  /** Double-click the element. */
  dblclick(options?: ActionOptions): void {
    this.bridge.call('element.dblclick', [this.elementId, options]);
  }

  /**
   * Fill the element with text (clears existing content first).
   * For inputs and textareas.
   */
  fill(value: string, options?: ActionOptions): void {
    this.bridge.call('element.fill', [this.elementId, value, options]);
  }

  /**
   * Type text into the element.
   * Waits for element to be visible, stable, receive events, enabled, and editable.
   */
  type(text: string, options?: ActionOptions): void {
    this.bridge.call('element.type', [this.elementId, text, options]);
  }

  /**
   * Press a key while the element is focused.
   * Supports key names ("Enter", "Tab") and combos ("Control+a").
   */
  press(key: string, options?: ActionOptions): void {
    this.bridge.call('element.press', [this.elementId, key, options]);
  }

  /** Clear the element's content (select all + delete). */
  clear(options?: ActionOptions): void {
    this.bridge.call('element.clear', [this.elementId, options]);
  }

  /** Check a checkbox (no-op if already checked). */
  check(options?: ActionOptions): void {
    this.bridge.call('element.check', [this.elementId, options]);
  }

  /** Uncheck a checkbox (no-op if already unchecked). */
  uncheck(options?: ActionOptions): void {
    this.bridge.call('element.uncheck', [this.elementId, options]);
  }

  /** Select an option in a <select> element by value. */
  selectOption(value: string, options?: ActionOptions): void {
    this.bridge.call('element.selectOption', [this.elementId, value, options]);
  }

  /** Hover over the element (move mouse to center, no click). */
  hover(options?: ActionOptions): void {
    this.bridge.call('element.hover', [this.elementId, options]);
  }

  /** Focus the element. */
  focus(options?: ActionOptions): void {
    this.bridge.call('element.focus', [this.elementId, options]);
  }

  /** Drag this element to a target element. */
  dragTo(target: ElementSync, options?: ActionOptions): void {
    this.bridge.call('element.dragTo', [this.elementId, (target as any).elementId, options]);
  }

  /** Tap the element (touch action). */
  tap(options?: ActionOptions): void {
    this.bridge.call('element.tap', [this.elementId, options]);
  }

  /** Scroll the element into view. */
  scrollIntoView(options?: ActionOptions): void {
    this.bridge.call('element.scrollIntoView', [this.elementId, options]);
  }

  /** Dispatch a DOM event on the element. */
  dispatchEvent(eventType: string, eventInit?: Record<string, unknown>, options?: ActionOptions): void {
    this.bridge.call('element.dispatchEvent', [this.elementId, eventType, eventInit, options]);
  }

  // --- State methods ---

  text(): string {
    const result = this.bridge.call<{ text: string }>('element.text', [this.elementId]);
    return result.text;
  }

  innerText(): string {
    const result = this.bridge.call<{ text: string }>('element.innerText', [this.elementId]);
    return result.text;
  }

  html(): string {
    const result = this.bridge.call<{ html: string }>('element.html', [this.elementId]);
    return result.html;
  }

  value(): string {
    const result = this.bridge.call<{ value: string }>('element.value', [this.elementId]);
    return result.value;
  }

  attr(name: string): string | null {
    const result = this.bridge.call<{ value: string | null }>('element.attr', [this.elementId, name]);
    return result.value;
  }

  getAttribute(name: string): string | null {
    return this.attr(name);
  }

  bounds(): BoundingBox {
    const result = this.bridge.call<{ box: BoundingBox }>('element.bounds', [this.elementId]);
    return result.box;
  }

  boundingBox(): BoundingBox {
    return this.bounds();
  }

  isVisible(): boolean {
    const result = this.bridge.call<{ visible: boolean }>('element.isVisible', [this.elementId]);
    return result.visible;
  }

  isHidden(): boolean {
    const result = this.bridge.call<{ hidden: boolean }>('element.isHidden', [this.elementId]);
    return result.hidden;
  }

  isEnabled(): boolean {
    const result = this.bridge.call<{ enabled: boolean }>('element.isEnabled', [this.elementId]);
    return result.enabled;
  }

  isChecked(): boolean {
    const result = this.bridge.call<{ checked: boolean }>('element.isChecked', [this.elementId]);
    return result.checked;
  }

  isEditable(): boolean {
    const result = this.bridge.call<{ editable: boolean }>('element.isEditable', [this.elementId]);
    return result.editable;
  }

  eval<T = unknown>(fn: string): T {
    const result = this.bridge.call<{ value: T }>('element.eval', [this.elementId, fn]);
    return result.value;
  }

  screenshot(): Buffer {
    const result = this.bridge.call<{ data: string }>('element.screenshot', [this.elementId]);
    return Buffer.from(result.data, 'base64');
  }

  waitUntil(state?: string, options?: { timeout?: number }): void {
    this.bridge.call('element.waitUntil', [this.elementId, state, options]);
  }

  setFiles(files: string[], options?: ActionOptions): void {
    this.bridge.call('element.setFiles', [this.elementId, files, options]);
  }

  role(): string {
    const result = this.bridge.call<{ role: string }>('element.role', [this.elementId]);
    return result.role;
  }

  label(): string {
    const result = this.bridge.call<{ label: string }>('element.label', [this.elementId]);
    return result.label;
  }

  find(selector: string | SelectorOptions, options?: { timeout?: number }): ElementSync {
    const result = this.bridge.call<{ elementId: number; info: ElementInfo }>('element.find', [this.elementId, selector, options]);
    return new ElementSync(this.bridge, result.elementId, result.info);
  }

  findAll(selector: string | SelectorOptions, options?: { timeout?: number }): ElementListSync {
    const result = this.bridge.call<{ listId: number; elementIds: number[]; count: number }>('element.findAll', [this.elementId, selector, options]);
    return new ElementListSync(this.bridge, result.listId);
  }
}
