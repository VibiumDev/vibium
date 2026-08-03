// Reports whether this machine pays the dead-virtual-GPU stall that
// clicker/bin/mtlshim.dylib exists to skip.
//
// Exit 0 means "affected": MTLCopyAllDevices was slow AND returned no devices.
// Both halves matter — a slow call that finds a real GPU must not be shimmed,
// because the shim hides the GPU rather than speeding it up.
//
// Built and run by `make mtlprobe`. See
// docs/how-to-guides/slow-chrome-launch-in-macos-vm.md

#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#import <stdio.h>
#import <sys/time.h>

// A healthy MTLCopyAllDevices returns in milliseconds; the affected guest takes
// ~15s. Anything past a second is the stall, not variance.
static const double kStallSeconds = 1.0;

static double now(void) {
  struct timeval t;
  gettimeofday(&t, NULL);
  return t.tv_sec + t.tv_usec / 1e6;
}

int main(void) {
  @autoreleasepool {
    double start = now();
    NSArray<id<MTLDevice>> *devices = MTLCopyAllDevices();
    double elapsed = now() - start;
    unsigned long count = (unsigned long)[devices count];

    int affected = (elapsed >= kStallSeconds && count == 0);
    fprintf(stderr, "MTLCopyAllDevices: %.2fs (%lu devices) -> %s\n",
            elapsed, count, affected ? "affected" : "healthy");
    return affected ? 0 : 1;
  }
}
