export { browser } from './browser';
export { Vibe, FindOptions, RecordingOptions } from './vibe';
export { Element, BoundingBox, ElementInfo, ActionOptions } from './element';

// Sync API
export { browserSync, VibeSync, ElementSync } from './sync';

// Error types
export {
  ConnectionError,
  TimeoutError,
  ElementNotFoundError,
  BrowserCrashedError,
} from './utils/errors';
