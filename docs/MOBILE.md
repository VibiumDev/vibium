# Mobile Automation with Vibium

Vibium supports native mobile app automation via **Appium**. This integration allows you to write Node.js scripts that interact with iOS and Android applications using a syntax similar to Vibium's browser automation.

## Architecture

Unlike the browser automation (which proxies through a Go-based `clicker` binary), the mobile client connects **directly** to an Appium server using the WebDriver HTTP protocol.

- **Client**: `clients/javascript` (Node.js SDK)
- **Server**: Appium (External dependency)

## Prerequisites

1.  **Node.js**: v18 or later.
2.  **Appium**: Install via `npm install -g appium`.
3.  **Drivers**: Install necessary drivers (e.g., `appium driver install xcuitest`).
4.  **Mobile Environment**: Xcode (iOS) or Android Studio (Android) configured properly.

## Setup

If you have forked this repository, follow these steps to build the Javascript client:

```bash
# 1. Navigate to the client directory
cd clients/javascript

# 2. Install dependencies
npm install

# 3. Build the SDK
npm run build
```

The build artifacts will be in `clients/javascript/dist`.

## Usage

You can import the `mobile` object from the built SDK to control a device.

### 1. Connect to Appium

```javascript
import { mobile } from "./clients/javascript/dist/index.mjs";

const device = await mobile.connect({
    url: "http://localhost:4723", // Default Appium URL
    capabilities: {
        platformName: "iOS",
        "appium:automationName": "XCUITest",
        "appium:deviceName": "iPhone 15",
        "appium:app": "/path/to/your.app",
        "appium:platformVersion": "17.2"
    }
});
```

### 2. Interactions

The `device` object provides methods to interact with the UI.

- **Tap**: `await device.tap("accessibility_id")`
- **Type**: `await device.type("accessibility_id", "text")`
- **Wait**: `await device.waitFor("accessibility_id", 10000)`
- **Source**: `await device.source()` (returns page XML)

### 3. Locator Strategies

By default, selectors are treated as **Accessibility IDs**. You can specify other strategies:

```javascript
// XPath
await device.tap("//XCUIElementTypeButton[@name='Save']", "xpath");

// iOS Class Chain
await device.tap("**/XCUIElementTypeButton[`label == 'Save'`]", "-ios class chain");
```

## Advanced Features

### Handling Permissions
System permission dialogs require a specific flow:
1.  Tap the button that triggers the dialog (e.g., "Enable Location").
2.  Wait for the dialog to appear.
3.  Tap "Allow".

```javascript
// Trigger the dialog
await device.tap("Enable Location");

// Wait for system dialog
await device.waitFor("Allow While Using App", 10000);
await device.tap("Allow While Using App");
```

### Secure Screens (Apple Pay / Passwords)
Some system screens (like Apple Pay or Password prompts) are **invisible** to XCUITest constraints for security reasons. They will not appear in `device.source()`.

**Workaround**: Use `tapCoordinates(x, y)` to perform a blind tap.

```javascript
try {
    // Try finding the element normally
    await device.tap("Pay with Passcode");
} catch (e) {
    // Fallback: Tap assumed location (e.g. bottom center)
    console.log("Element not found (secure window?), tapping coordinates.");
    await device.tapCoordinates(200, 750); 
}
```

### Geolocation
Set the device's location (requires the app to be authorized for location).

```javascript
await device.setGeoLocation({
    latitude: 40.7128,
    longitude: -74.0060
});
```

## Troubleshooting

1.  **Element Not Found**: 
    - Use `await device.source()` and print/save the XML to see what Appium actually sees.
    - Check for hidden characters like Non-Breaking Spaces (`\u00A0`) in button labels (common in iOS).
2.  **App Not Installing**: 
    - Ensure `appium:app` path is absolute.
    - Check `appium:fullReset` capability if you need a clean install.
3.  **Secure Windows**:
    - If `source()` does not show the element, you cannot select it by ID/Text. Use `tapCoordinates`.
