package com.vibium.process;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

public class PlatformTest {
    @Test
    public void npmPlatformIsOneOfExpectedValues() {
        String p = Platform.npmPlatform();
        assertTrue(p.equals("win32") || p.equals("darwin") || p.equals("linux"), "unexpected platform: " + p);
    }

    @Test
    public void npmArchIsOneOfExpectedValues() {
        String a = Platform.npmArch();
        assertTrue(a.equals("x64") || a.equals("arm64"), "unexpected arch: " + a);
    }
}

