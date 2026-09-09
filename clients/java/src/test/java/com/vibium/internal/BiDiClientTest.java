package com.vibium.internal;

import com.vibium.errors.ElementNotFoundException;
import com.vibium.errors.VibiumException;
import com.vibium.errors.VibiumTimeoutException;
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

    // #483: classification must not depend on wording the client does not
    // own. A page's own error text containing "not found" is not an element
    // lookup failure.

    @Test
    void scriptErrorMentioningNotFoundIsNotAnElementError() {
        VibiumException e = BiDiClient.mapError("error",
            "eval failed: script exception: Error: widget not found in registry");
        assertEquals(VibiumException.class, e.getClass());
    }

    @Test
    void scriptErrorMentioningNoElementsIsNotAnElementError() {
        VibiumException e = BiDiClient.mapError("error",
            "eval failed: script exception: Error: no elements match the filter");
        assertEquals(VibiumException.class, e.getClass());
    }

    @Test
    void realElementFailureStillMapsThroughTheTimeoutBranch() {
        VibiumException e = BiDiClient.mapError("timeout",
            "timeout after 30s waiting for '#gone': element not found");
        assertEquals(ElementNotFoundException.class, e.getClass());
    }

    @Test
    void elementNotFoundPhraseOutsideTimeoutStillMaps() {
        VibiumException e = BiDiClient.mapError("error",
            "element not found");
        assertEquals(ElementNotFoundException.class, e.getClass());
    }

    @Test
    void plainTimeoutStaysATimeout() {
        VibiumException e = BiDiClient.mapError("timeout",
            "timeout after 30s waiting for network idle");
        assertEquals(VibiumTimeoutException.class, e.getClass());
    }
}
