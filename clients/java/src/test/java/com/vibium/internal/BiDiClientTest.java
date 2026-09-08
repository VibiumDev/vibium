package com.vibium.internal;

import com.vibium.errors.ElementNotFoundException;
import org.junit.jupiter.api.Test;

import java.util.Arrays;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class BiDiClientTest {

    @Test
    void replacesReceiverFramesWithCommandCallerFrames() {
        ElementNotFoundException error = new ElementNotFoundException("element not found");

        attachFromUserCall(error);

        assertTrue(Arrays.stream(error.getStackTrace())
            .anyMatch(frame -> frame.getMethodName().equals("replacesReceiverFramesWithCommandCallerFrames")));
        assertFalse(Arrays.stream(error.getStackTrace())
            .anyMatch(frame -> frame.getClassName().equals(BiDiClient.class.getName())));
        assertEquals("element not found", error.getMessage());
    }

    private void attachFromUserCall(ElementNotFoundException error) {
        StackTraceElement[] callerStack = BiDiClient.captureCallerStack();
        BiDiClient.attachCallerStack(error, callerStack);
    }
}
