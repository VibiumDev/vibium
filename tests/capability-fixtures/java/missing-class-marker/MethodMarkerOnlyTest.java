package com.vibium.engine;

import org.junit.jupiter.api.Test;

// Regex fixture only, never compiled: a method-level marker without the
// class-level baseline must be rejected.
class MethodMarkerOnlyTest {
    @RequiresCapability("core")
    @Test
    void marked() {}
}
