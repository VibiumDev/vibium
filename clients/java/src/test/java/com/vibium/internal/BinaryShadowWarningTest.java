package com.vibium.internal;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * A PATH install shadowing the jar's packaged binary must be called out when
 * the versions differ, and stay silent when they match or cannot be compared
 * (#331).
 */
class BinaryShadowWarningTest {

    @Test
    void differingVersionsProduceAWarningNamingBoth() {
        String w = BinaryResolver.shadowWarning("/usr/local/bin/vibium", "26.5.31", "26.8.21");
        assertNotNull(w);
        assertTrue(w.contains("26.5.31"), "warning should name the PATH binary's version");
        assertTrue(w.contains("26.8.21"), "warning should name the jar's version");
        assertTrue(w.contains("/usr/local/bin/vibium"), "warning should name the binary that runs");
        assertTrue(w.contains("VIBIUM_BIN_PATH"), "warning should say how to override");
    }

    @Test
    void matchingVersionsStaySilent() {
        assertNull(BinaryResolver.shadowWarning("/usr/local/bin/vibium", "26.8.21", "26.8.21"));
    }

    @Test
    void unreadablePathVersionStaysSilent() {
        assertNull(BinaryResolver.shadowWarning("/usr/local/bin/vibium", null, "26.8.21"));
    }

    @Test
    void parseVersionHandlesTypicalOutput() {
        assertEquals("26.8.21", BinaryResolver.parseVersion("vibium v26.8.21"));
        assertEquals("26.8.21", BinaryResolver.parseVersion("vibium 26.8.21"));
        assertEquals("26.8.21", BinaryResolver.parseVersion("vibium v26.8.21\nextra line"));
    }

    @Test
    void parseVersionReturnsNullWhenThereIsNoNumber() {
        assertNull(BinaryResolver.parseVersion(""));
        assertNull(BinaryResolver.parseVersion("command not found"));
        assertNull(BinaryResolver.parseVersion(null));
    }
}
