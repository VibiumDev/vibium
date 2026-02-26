/** Sync route handler data — plain object representing the intercepted request. */
export interface RouteRequest {
  url: string;
  method: string;
  headers: Record<string, string>;
  postData: string | null;
}

/** Decision returned by a sync route handler. */
export interface RouteDecision {
  action: 'fulfill' | 'continue' | 'abort';
  status?: number;
  headers?: Record<string, string>;
  contentType?: string;
  body?: string;
  url?: string;
  method?: string;
  postData?: string;
}

/**
 * Sync wrapper for an intercepted network request.
 * The user's handler calls fulfill(), continue(), or abort() to set the decision.
 * If none is called, the default is 'continue'.
 */
export class RouteSync {
  readonly request: RouteRequest;
  /** @internal */
  _decision: RouteDecision = { action: 'continue' };

  constructor(request: RouteRequest) {
    this.request = request;
  }

  /** Fulfill the request with a custom response. */
  fulfill(response: {
    status?: number;
    headers?: Record<string, string>;
    contentType?: string;
    body?: string;
  } = {}): void {
    this._decision = { action: 'fulfill', ...response };
  }

  /** Continue the request, optionally with overrides. */
  continue(overrides?: {
    url?: string;
    method?: string;
    headers?: Record<string, string>;
    postData?: string;
  }): void {
    this._decision = { action: 'continue', ...overrides };
  }

  /** Abort the request. */
  abort(): void {
    this._decision = { action: 'abort' };
  }
}
