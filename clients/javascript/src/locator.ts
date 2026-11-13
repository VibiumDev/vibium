import { readFile } from 'fs/promises';
import type { ElementInfo, SelectorOptions } from './element';

/** The find result fields every surface returns. */
export interface FoundElement {
  tag: string;
  text: string;
  box: { x: number; y: number; width: number; height: number };
  ref?: string;
  role?: string;
  name?: string;
  source?: string;
  selector?: string;
  details?: Record<string, unknown>;
  index?: number;
}

/** Encode a reference image for the wire: a path is read, bytes are base64'd, a data: URL passes through. */
export async function encodeImage(image: string | Uint8Array): Promise<string> {
  if (typeof image !== 'string') return Buffer.from(image).toString('base64');
  if (image.startsWith('data:')) return image;
  return (await readFile(image)).toString('base64');
}

/**
 * Wire params for a locator. A bare string is a description. css becomes
 * selector, the name element commands read, so one locator object serves
 * both the find and the actions that follow it.
 */
export async function locatorParams(selector: string | SelectorOptions): Promise<Record<string, unknown>> {
  if (typeof selector === 'string') return { description: selector, bare: true };
  const { image, css, ...rest } = selector;
  const params: Record<string, unknown> = { ...rest };
  if (css !== undefined) params.selector = css;
  if (image !== undefined) params.image = await encodeImage(image);
  return params;
}

/**
 * Whether later commands should address the match by what find resolved
 * rather than by re-running the locator: a description or image would run
 * the search again, and tag is not an action locator.
 */
export function resolvedHandle(selector: string | SelectorOptions): boolean {
  if (typeof selector === 'string') return true;
  return selector.description !== undefined || selector.image !== undefined || selector.tag !== undefined || selector.name !== undefined;
}

/** The selector an element is addressed by after find: its DOM selector, or its @ref for a visual match. */
export function handleSelector(result: FoundElement): string {
  return result.selector || result.ref || '';
}

export function infoFrom(result: FoundElement): ElementInfo {
  const info: ElementInfo = { tag: result.tag, text: result.text, box: result.box };
  if (result.ref !== undefined) info.ref = result.ref;
  if (result.role !== undefined) info.role = result.role;
  if (result.name !== undefined) info.name = result.name;
  if (result.source !== undefined) info.source = result.source;
  if (result.selector !== undefined) info.selector = result.selector;
  if (result.details !== undefined) info.details = result.details;
  return info;
}

/** The css selector of a locator, which stays the element's own handle so scoped finds and re-resolution see it. */
export function cssOf(params: Record<string, unknown>): string {
  return typeof params.selector === 'string' ? params.selector : '';
}

/** Strip per-call options and the encoded image from a locator before it is reused by element commands. */
export function reusableParams(params: Record<string, unknown>): Record<string, unknown> {
  const out = { ...params };
  for (const key of ['context', 'timeout', 'image', 'details', 'strategy', 'provider', 'model', 'aiBaseURL', 'reasoningEffort', 'limit', 'bare']) {
    delete out[key];
  }
  return out;
}
