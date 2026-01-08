/**
 * JS Library Tests: Video Recording
 * Tests startRecording() and stopRecording() methods
 *
 * Note: These tests require FFmpeg to be installed on the system.
 * If FFmpeg is not available, tests will be skipped.
 */

const { test, describe, before } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const { execSync } = require('node:child_process');

// Import from built library
const { browser } = require('../../clients/javascript/dist');

// Check if FFmpeg is available
function isFFmpegAvailable() {
  try {
    execSync('ffmpeg -version', { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

const skipReason = isFFmpegAvailable() ? null : 'FFmpeg not available';

describe('JS Recording API', { skip: skipReason }, () => {
  test('startRecording() and stopRecording() create video file', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');

      // Start recording
      await vibe.startRecording({ fps: 5, format: 'mp4' });

      // Wait a bit and navigate to capture some frames
      await new Promise(resolve => setTimeout(resolve, 1000));

      // Navigate to another page
      await vibe.go('https://the-internet.herokuapp.com/add_remove_elements/');

      // Wait a bit more
      await new Promise(resolve => setTimeout(resolve, 1000));

      // Stop recording
      const videoPath = await vibe.stopRecording();

      // Verify video was created
      assert.ok(videoPath, 'Should return video path');
      assert.ok(fs.existsSync(videoPath), 'Video file should exist');

      // Check file has content
      const stats = fs.statSync(videoPath);
      assert.ok(stats.size > 1000, 'Video file should have reasonable size');

      // Clean up
      fs.unlinkSync(videoPath);
    } finally {
      await vibe.quit();
    }
  });

  test('startRecording() with custom output path', async () => {
    const outputPath = path.join(__dirname, 'test-recording.mp4');

    // Clean up any previous test file
    if (fs.existsSync(outputPath)) {
      fs.unlinkSync(outputPath);
    }

    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');

      // Start recording with custom path
      await vibe.startRecording({ fps: 5, format: 'mp4', outputPath });

      // Wait to capture frames
      await new Promise(resolve => setTimeout(resolve, 1500));

      // Stop recording
      const returnedPath = await vibe.stopRecording();

      // Verify path matches
      assert.strictEqual(returnedPath, outputPath, 'Should return specified output path');
      assert.ok(fs.existsSync(outputPath), 'Video file should exist at specified path');

      // Clean up
      fs.unlinkSync(outputPath);
    } finally {
      await vibe.quit();
    }
  });

  test('stopRecording() without starting throws error', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');

      // Try to stop without starting
      await assert.rejects(
        async () => await vibe.stopRecording(),
        /no recording in progress/i,
        'Should throw error when no recording is in progress'
      );
    } finally {
      await vibe.quit();
    }
  });

  test('startRecording() twice throws error', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');

      // Start first recording
      await vibe.startRecording({ fps: 5 });

      // Try to start another
      await assert.rejects(
        async () => await vibe.startRecording({ fps: 5 }),
        /recording already in progress/i,
        'Should throw error when recording already in progress'
      );

      // Clean up: stop the first recording
      const videoPath = await vibe.stopRecording();
      if (fs.existsSync(videoPath)) {
        fs.unlinkSync(videoPath);
      }
    } finally {
      await vibe.quit();
    }
  });
});
