# Slow Chrome Launch in a macOS VM

On a macOS guest whose virtual GPU doesn't work, **every Chrome launch costs an
extra ~15 seconds**. Since `make test` launches Chrome ~120 times, this is the
difference between a ~6 minute suite and an ~18 minute one.

This guide explains the cause, how to confirm you're affected, and an opt-in
workaround. It applies only to VM guests with a broken/absent GPU — on real
hardware, and on CI, none of this is relevant.

## Symptom

Every browser-owning test file takes a flat ~16–20s regardless of what it
asserts, while test files that reuse an already-open browser run in 1–5s.

## Confirming it

The stall is not in Chrome, chromedriver, or vibium. It reproduces with a
four-line Metal program:

```objc
#import <Metal/Metal.h>
#import <stdio.h>
#import <sys/time.h>
static double now(){struct timeval t;gettimeofday(&t,0);return t.tv_sec+t.tv_usec/1e6;}
int main(){
  double s=now();
  NSArray<id<MTLDevice>> *d = MTLCopyAllDevices();
  printf("MTLCopyAllDevices: %.2fs (%lu devices)\n", now()-s, (unsigned long)[d count]);
  return 0;
}
```

```bash
clang -fobjc-arc -framework Metal -framework Foundation -o /tmp/mtl /tmp/mtl.m
/tmp/mtl
```

Affected guest:

```
MTLCopyAllDevices: 15.01s (0 devices)
```

Healthy machine: well under a second, with at least one device.

## Cause

Chrome — even headless — creates an AppKit `NSWindow`, which makes macOS
enumerate Metal GPU devices:

```
NSWindow initWithContentRect:  (AppKit)
  -> _NXCreateWindow -> NSCGSWindow _createContext
    -> MTLCopyAllDevices  (Metal)
      -> MTLIOAccelService initWithAcceleratorPort:
        -> AppleParavirtGPUMetalIOGPUFamily
          -> IOGPUDeviceCreateWithOptions -> IOServiceOpen
            -> mach_msg2_trap            <- blocked ~15s
```

The guest advertises an `AppleParavirtGPU` IOService that never answers.
`IOServiceOpen` blocks for a fixed ~15s, then fails with
`kIOReturnUnsupported` (`0xE00002C7`) and reports zero devices. The kext is
loaded but matched nothing (`kextstat` shows refcount 0), and
`system_profiler SPDisplaysDataType` is empty.

Chrome then proceeds correctly with no GPU. **The 15s buys nothing** — it is
purely the time taken to discover there is no GPU.

### What does not fix it

Checked and ruled out, all still ~16.6s:

- `--disable-gpu`, `--headless=old`, `--use-gl=disabled`, `--in-process-gpu`
  alone, `--use-angle=swiftshader` (`--use-angle=metal` is worse, ~31s)
- Switching VM host apps. UTM's Apple backend already *is*
  Virtualization.framework; `VZMacGraphicsDeviceConfiguration` exposes only
  display geometry, with no GPU flag for another app to set differently.
- UTM settings. There is no GPU toggle for macOS guests
  ([utmapp/UTM#5785](https://github.com/utmapp/UTM/issues/5785)), and the
  display view is already correctly attached.
- Unloading the unused kext so Metal fails fast — blocked by SIP.

## Workaround

A `DYLD_INSERT_LIBRARIES` dylib that short-circuits Metal device lookup:

| Configuration | Launch |
| --- | --- |
| baseline | 16.6s |
| shim covering 2 entry points | 15.3s |
| **shim covering all 3 entry points** | **1.7s** |

Three entry points must be covered, not two. The browser process reaches the
dead driver through `MTLCopyAllDevices` / `MTLCreateSystemDefaultDevice`, but
Chrome's **GPU process** uses `MTLCopyAllDevicesWithObserver`. Miss that one
and the GPU process still stalls ~15s, holding up the browser behind it.

Do **not** reach for Chrome process-model flags to work around a missed entry
point. `--in-process-gpu` hides the GPU-process stall and appears to work, but
it collapses the GPU service into the browser process — so any GPU stall wedges
the browser, and CLI commands start timing out mid-suite. `--single-process`
is worse still. Cover the entry point instead.

### The shim

```objc
// Short-circuit Metal device lookup in VMs whose paravirt GPU makes it block
// ~15s before reporting no GPU anyway. Same result, without the wait.
#import <Foundation/Foundation.h>
#import <Metal/Metal.h>

static NSArray *shim_MTLCopyAllDevices(void) {
  return [[NSArray alloc] init];        // +1, matching the Copy convention
}
static id<MTLDevice> shim_MTLCreateSystemDefaultDevice(void) {
  return nil;                            // what it returns anyway, 15s sooner
}

#define INTERPOSE(_new, _old)                                                  \
  __attribute__((used)) static struct { const void *r; const void *o; }        \
  _interpose_##_old __attribute__((section("__DATA,__interpose"))) = {         \
    (const void *)(uintptr_t)&_new, (const void *)(uintptr_t)&_old };

INTERPOSE(shim_MTLCopyAllDevices, MTLCopyAllDevices)
INTERPOSE(shim_MTLCreateSystemDefaultDevice, MTLCreateSystemDefaultDevice)
```

```bash
clang -fno-objc-arc -dynamiclib -framework Metal -framework Foundation \
  -o /tmp/mtlshim.dylib /tmp/mtlshim.m
```

Both replaced functions return exactly what the real ones return on an
affected guest — an empty array and `nil`. The shim changes *when* Chrome gets
that answer, not *what* the answer is. Verify with the probe above before
trusting it: if `MTLCopyAllDevices` reports any devices, the shim is a lie and
must not be used.

### What actually happens, in order

The shim launches nothing. It has no `main()` and no process-spawning code —
it is passive cargo, unrelated to vibium, and not Chrome-specific. vibium still
does all the launching:

```
vibium (Go) --spawns--> chromedriver --spawns--> Chrome
                                                   |
                                    dyld loads mtlshim.dylib into it
                                    ONLY IF DYLD_INSERT_LIBRARIES is set
```

1. Something sets `DYLD_INSERT_LIBRARIES=/tmp/mtlshim.dylib` in the environment.
2. Chrome gets spawned — by chromedriver, by vibium, or from a shell.
3. **dyld** (macOS's dynamic linker, not the shim) sees the variable while
   loading Chrome and maps `mtlshim.dylib` into Chrome's address space.
4. dyld rewrites two symbol bindings, via the `__DATA,__interpose` section, so
   Metal calls land on the shim's functions.
5. Chrome runs normally. When its window setup calls `MTLCopyAllDevices()`, it
   gets `[]` immediately instead of after 15s.
6. Chrome exits; the shim goes with it.

The shim's code runs only if something calls those two functions. Otherwise it
sits inert.

### Using it

Measured on an affected guest, running the process suite directly:

```bash
DYLD_INSERT_LIBRARIES=/tmp/mtlshim.dylib VIBIUM_VM_FAST_LAUNCH=1 \
  node --test --test-timeout=30000 --test-force-exit --test-concurrency=1 \
  tests/js/async/process.test.js tests/js/sync/process.test.js
```

91s -> 16s, all tests passing. A full vibium launch (`vibium bidi-test`,
covering chromedriver startup and the BiDi handshake) goes 18.1s -> 3.1s.

**Exporting the variable and running `make` does not work.** macOS strips
`DYLD_*` when exec'ing a SIP-protected binary, and make runs every recipe
through `/bin/sh`:

```bash
# stripped — sh is exec'd with the variable already set
DYLD_INSERT_LIBRARIES=/tmp/mtlshim.dylib sh -c 'node -e "..."'   # (STRIPPED)

# survives — sh sets it for its child, and node is ad-hoc signed
sh -c 'DYLD_INSERT_LIBRARIES=/tmp/mtlshim.dylib node -e "..."'   # /tmp/mtlshim.dylib
```

So it has to be assigned *inline on the command* inside the recipe, the same
way the Makefile already passes `VIBIUM_BIN_PATH`. Setting it on the Chrome
`exec.Cmd` in `clicker/internal/browser/launcher.go` avoids the problem
entirely and is narrower besides.

## Cautions

- **Never enable on real hardware.** Where a GPU exists, the shim doesn't
  shortcut to the same answer — it hides the GPU, which would break rendering
  and mask genuine GPU behavior. It is correct *only* because this guest has no
  working GPU.
- **Never ship it.** It must not reach published binaries or packages. vibium
  does nothing unless `VIBIUM_VM_FAST_LAUNCH` points at a dylib, so this is
  opt-in by construction.
- **Decide by probe, not by environment.** The criterion is whether
  `MTLCopyAllDevices` is slow and reports zero devices *on that machine* — not
  whether it's a laptop or CI. Today's CI is Linux, where the flag is inert and
  the shim irrelevant. If macOS CI is added later, run the probe on the runner:
  if it's affected, enabling this is legitimate and buys the same ~3x.
- **Chrome's process model is unchanged**, so local runs exercise the same
  topology as CI and as users. Keep it that way: if a stall reappears, extend
  the shim to cover the entry point rather than reaching for a flag that
  restructures Chrome's processes.
- **Environment variables inherit down the whole process tree.** Exporting
  `DYLD_INSERT_LIBRARIES` before `make test` loads the dylib into node, python,
  the Go binary, chromedriver, and the JVMs too. Harmless — none of them call
  Metal — but wider than the problem needs. Setting it on just the Chrome
  `exec.Cmd` in `clicker/internal/browser/launcher.go` is the surgical option.
- **Coverage is enumerated, not general.** Three entry points are interposed.
  If a future Chrome or macOS reaches the driver another way, launches go slow
  again — sample the stalled process to find the new entry point and add it.
- **Unsupported territory.** Interposing framework internals isn't a contract
  Apple maintains. A macOS update could change which entry point is hit first
  and the shim would silently stop helping. The failure mode is benign — slow
  launches return, nothing breaks.
- `DYLD_INSERT_LIBRARIES` works here only because Chrome for Testing is
  ad-hoc/linker-signed with no hardened runtime (`codesign -dv` shows
  `flags=0x20002`). A hardened-runtime Chrome would strip it.

## Alternatives that need no hack

- **Run `make test` on the host** instead of in the VM. Lands near CI timings
  and sidesteps all of the above.
- **Reduce launch count.** Test files that share a browser are ~10x cheaper.
  `test-daemon` spends roughly 225s of its 275s launching, because each suite
  stops and restarts the daemon.
- **Raise `JS_PARALLEL` / `PY_PARALLEL` / `JAVA_PARALLEL`.** The stall is a
  blocked `mach_msg` consuming no CPU, so concurrency is cheaper here than the
  conservative defaults assume.
