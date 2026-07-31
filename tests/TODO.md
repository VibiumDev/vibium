# Test Improvements TODO

Tests are functional but could be improved:

## Coverage Gaps
- [ ] `element.getAttribute()` not tested
- [ ] `element.boundingBox()` not tested

## Auto-Wait Tests
- [ ] "click waits for actionable" should test element that starts hidden and becomes visible
- [ ] Remove hardcoded `setTimeout` in async-api.test.js (line 85) - rely on auto-wait

## Negative Cases
- [ ] Test clicking element that disappears after find()
- [ ] Test type() on non-editable element
- [ ] Test click() on disabled element

## Performance
- [ ] Share browser instances within describe blocks to reduce ~1s overhead per test
- [ ] Run tests in parallel (currently sequential due to resource exhaustion issues)
