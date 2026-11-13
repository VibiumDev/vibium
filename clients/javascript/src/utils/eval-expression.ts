/**
 * evaluate() accepts a JavaScript expression string. The docs and Quick Start
 * also show the arrow-function form, `evaluate(() => document.title)`, which is
 * what most people reach for. That used to resolve to null silently: a function
 * is dropped by JSON.stringify on the way to the binary, so the expression
 * arrived empty and the page was never touched (#639).
 *
 * toEvalExpression turns a function back into something that runs: its source,
 * wrapped and invoked, so `() => document.title` becomes `(() => document.title)()`.
 * Strings pass through unchanged.
 *
 * The function body runs in the page, not the client, so it cannot close over
 * client-side variables or take arguments — the same rule as every other
 * page-evaluation API. A function that references an outer variable throws a
 * ReferenceError in the page, which surfaces as an evaluation error instead of
 * the old silent null.
 */
export function toEvalExpression(expression: string | (() => unknown)): string {
  if (typeof expression === 'function') {
    return `(${expression.toString()})()`;
  }
  return expression;
}
