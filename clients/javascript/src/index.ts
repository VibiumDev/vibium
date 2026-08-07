export { browser, firefox, chrome, Browser, StartOptions } from './browser';
export { Page, Keyboard, Mouse, Touch, ScreenshotOptions, A11yNode, FindOptions } from './page';
export { Clock, ClockInstallOptions } from './clock';
export { BrowserContext, Cookie, SetCookieParam, StorageState, OriginState } from './context';
export { Recording, RecordingStartOptions, RecordingStopOptions } from './recording';
export { Screencast, ScreencastStartOptions, ScreencastStopOptions } from './screencast';
export { Element, BoundingBox, ElementInfo, ActionOptions, SelectorOptions, FluentElement, fluent } from './element';
export { Route } from './route';
export { Request, Response } from './network';
export { Dialog } from './dialog';
export { ConsoleMessage } from './console';
export { Download } from './download';
export { WebSocketInfo } from './websocket';

// Sync API — import from 'vibium/sync' for the sync browser launcher
export {
  BrowserSync,
  PageSync,
  ElementSync,
  KeyboardSync,
  MouseSync,
  TouchSync,
  ClockSync,
  BrowserContextSync,
  RecordingSync,
  ScreencastSync,
} from './sync';

// Error types
export {
  ConnectionError,
  TimeoutError,
  ElementNotFoundError,
  BrowserCrashedError,
} from './utils/errors';
