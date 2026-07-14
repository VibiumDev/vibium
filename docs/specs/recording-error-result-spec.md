# Spec: Add `error` and `result` to Recording `after` Events

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

## Context

Playwright's trace format includes `error` and `result` fields on `after` events. The trace viewer uses these to show red error markers on failed actions and display return values for queries like `.text()`. Vibium currently records `after` events with only `callId`, `endTime`, and `afterSnapshot` — no visibility into whether actions succeeded or what they returned.

## Two recording paths

- **API path** (`router.go:dispatch()`) — handlers call `sendSuccess()`/`sendError()` directly and return void. `dispatch()` has no visibility into the outcome.
- **Agent/MCP path** (`handlers.go:Call()`) — returns `(*ToolsCallResult, error)`, both available but not passed to `RecordActionEnd()`.

## Files to Modify

1. `clicker/internal/api/recording.go` — extend `RecordActionEnd` signature
2. `clicker/internal/api/router.go` — stash result/error on session, pass to `RecordActionEnd`
3. `clicker/internal/agent/handlers.go` — pass result/error to `RecordActionEnd`
4. `docs/explanation/recording-format.md` — document new fields

## Changes

### 1. `recording.go` — Extend `RecordActionEnd`

Add `result interface{}` and `actionErr error` parameters:

```go
func (t *Recorder) RecordActionEnd(callId, afterSnapshot string, endTime time.Time, box *BoxInfo, result interface{}, actionErr error)
```

In the `after` event map:
- If `actionErr != nil` → `"error": map[string]interface{}{"message": actionErr.Error()}`
- If `result != nil` → `"result": result`

### 2. `router.go` — Capture result/error in API path

Add two fields to `BrowserSession` (alongside existing `lastElementBox`):
```go
lastResult interface{} // stashed by sendSuccess for recording
lastError  error       // stashed by sendError for recording
```

In `sendSuccess()`: stash `result` on session under `session.mu`.
In `sendError()`: stash `err` on session under `session.mu`.

In `dispatch()`, in the same block that reads `lastElementBox`, also read+clear `lastResult`/`lastError`, then pass them to `RecordActionEnd`.

### 3. `handlers.go` — Pass result/error in MCP path

`Call()` already has `result` and `err`. Extract the text content from `result.Content[0].Text` (if available) and pass both to `RecordActionEnd`.

### 4. `recording-format.md` — Document new fields

Add to the `after` event field table:

| Field | Type | Description |
|-------|------|-------------|
| `error` | object | `{message: string}` — present when the action failed |
| `result` | any | Action return value — present when the action succeeded with a meaningful result |

## Verification

1. `make test` — all tests pass
2. Inspect a recording zip: successful actions have `result`, failed actions have `error`
