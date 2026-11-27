package com.vibium;

import com.vibium.types.FulfillOptions;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * The fulfilled status must reach the wire under "statusCode", the key the
 * engine reads, as the JS and Python clients send it. Sending "status"
 * left every mocked status code ignored (#854).
 */
class FulfillOptionsTest {

    @Test
    void statusIsSentUnderStatusCode() {
        Map<String, Object> params = new FulfillOptions().status(404).body("nope").toParams();
        assertEquals(404, params.get("statusCode"), "status must be sent as statusCode");
        assertFalse(params.containsKey("status"), "the ignored 'status' key must not be sent");
        assertEquals("nope", params.get("body"));
    }

    @Test
    void statusOmittedWhenUnset() {
        Map<String, Object> params = new FulfillOptions().body("ok").toParams();
        assertFalse(params.containsKey("statusCode"), "statusCode should stay unset");
    }
}
