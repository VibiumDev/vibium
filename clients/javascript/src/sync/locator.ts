import type { SelectorOptions } from '../element';

/** Image bytes do not survive the worker bridge as bytes; send them as a data: URL. */
export function bridgeLocator(selector: string | SelectorOptions): string | SelectorOptions {
  if (typeof selector === 'string' || !(selector.image instanceof Uint8Array)) return selector;
  const mime = selector.image[0] === 0xff ? 'image/jpeg' : 'image/png';
  return { ...selector, image: `data:${mime};base64,` + Buffer.from(selector.image).toString('base64') };
}
