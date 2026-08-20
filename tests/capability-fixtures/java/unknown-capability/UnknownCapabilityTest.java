package com.vibium.engine;

import org.junit.jupiter.api.Test;

// Regex fixture only, never compiled: a capability missing from the manifest
// must be rejected.
@RequiresCapability("nonexistent")
class UnknownCapabilityTest {
    @Test
    void marked() {}
}
