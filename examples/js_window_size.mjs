// Example: Launch browser with custom window size

import { browser } from 'vibium';

// Launch with 1920x1080 resolution
const vibe = await browser.launch({ width: 1920, height: 1080 });

await vibe.go('https://example.com');

// Take a screenshot at the specified resolution
const png = await vibe.screenshot();
console.log(`Screenshot captured: ${png.length} bytes`);

await vibe.quit();

console.log('Done! Browser launched with 1920x1080 resolution');
