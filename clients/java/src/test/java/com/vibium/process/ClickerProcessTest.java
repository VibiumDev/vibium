package com.vibium.process;

import org.junit.jupiter.api.Test;

import java.util.OptionalInt;

import static org.junit.jupiter.api.Assertions.*;

public class ClickerProcessTest {
    @Test
    public void parsesPortFromServerListeningLine() {
        OptionalInt port = ClickerProcess.parsePortFromLine("Server listening on ws://localhost:9515");
        assertTrue(port.isPresent());
        assertEquals(9515, port.getAsInt());
    }

    @Test
    public void ignoresUnrelatedOutput() {
        OptionalInt port = ClickerProcess.parsePortFromLine("some other line");
        assertFalse(port.isPresent());
    }
}
