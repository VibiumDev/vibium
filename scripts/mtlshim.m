// Returns what Metal already returns on a macOS VM guest with a dead virtual
// GPU — nothing — without its ~15s wait. Never use where a GPU works.
// Built by `make mtlshim`. See docs/how-to-guides/slow-chrome-launch-in-macos-vm.md

#import <Foundation/Foundation.h>
#import <Metal/Metal.h>

static NSArray *shim_MTLCopyAllDevices(void) {
  return [[NSArray alloc] init];        // +1, matching the Copy convention
}

static id<MTLDevice> shim_MTLCreateSystemDefaultDevice(void) {
  return nil;                            // what it returns anyway, 15s sooner
}

// Chrome's GPU process reaches the same dead driver through this one.
static NSArray *shim_MTLCopyAllDevicesWithObserver(
    id<NSObject> __strong *observer, MTLDeviceNotificationHandler handler) {
  if (observer) *observer = nil;
  return [[NSArray alloc] init];
}

#define INTERPOSE(_new, _old)                                                  \
  __attribute__((used)) static struct { const void *r; const void *o; }        \
  _interpose_##_old __attribute__((section("__DATA,__interpose"))) = {         \
    (const void *)(uintptr_t)&_new, (const void *)(uintptr_t)&_old };

INTERPOSE(shim_MTLCopyAllDevices, MTLCopyAllDevices)
INTERPOSE(shim_MTLCreateSystemDefaultDevice, MTLCreateSystemDefaultDevice)
INTERPOSE(shim_MTLCopyAllDevicesWithObserver, MTLCopyAllDevicesWithObserver)
