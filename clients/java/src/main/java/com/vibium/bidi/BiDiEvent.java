package com.vibium.bidi;

import com.google.gson.JsonObject;

public final class BiDiEvent {
    public final String method;
    public final JsonObject params;

    public BiDiEvent(String method, JsonObject params) {
        this.method = method;
        this.params = params;
    }
}

