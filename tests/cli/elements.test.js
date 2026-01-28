/**
 * CLI Tests: Element Finding, Click, and Type
 * Tests the clicker binary directly
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const path = require('node:path');

const CLICKER = path.join(__dirname, '../../clicker/bin/clicker');

describe('CLI: Elements', () => {
  test('find command locates element', () => {
    const result = execSync(`${CLICKER} find https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /tag=a/i, 'Should find anchor tag');
    // Link text may be "More information..." or "Learn more" depending on page version
    assert.match(result, /(More information|Learn more)/i, 'Should show link text');
    assert.match(result, /box=/i, 'Should show bounding box');
  });

  test('click command navigates via link', () => {
    const result = execSync(`${CLICKER} click https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /iana\.org/i, 'Should navigate to IANA after clicking link');
  });

  test('type command enters text into input', () => {
    const result = execSync(
      `${CLICKER} type https://the-internet.herokuapp.com/inputs "input" "12345"`,
      {
        encoding: 'utf-8',
        timeout: 30000,
      }
    );
    assert.match(result, /12345/, 'Should show typed text in result');
  });

  test('type command enters text with carriage returns into textarea', () => {
    const result = execSync(
      `${CLICKER} type https://seleniumbase.io/demo_page "textarea" $'123\\r456'`,
      {
        encoding: 'utf-8',
        timeout: 30000,
        shell: '/bin/bash',
      }
    );
    // The output should show the textarea value with a newline (via carriage return)
    assert.match(result, /value is now: 123\n456/, 'Should show typed value with newline (via carriage return)');
  });

  test('type command enters text with newlines into textarea', () => {
    const result = execSync(
      `${CLICKER} type https://seleniumbase.io/demo_page "textarea" $'123\\n456'`,
      {
        encoding: 'utf-8',
        timeout: 30000,
        shell: '/bin/bash',
      }
    );
    // The output should show the textarea value with a newline
    assert.match(result, /value is now: 123\n456/, 'Should show typed value with newline');
  });
});
