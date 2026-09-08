package com.vibium.internal;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.vibium.types.CheckOptions;
import com.vibium.types.CheckResult;

/** Existing semantic command; provider configuration is read in the Go runtime. */
public final class Check {
    private Check() {}
    public static CheckResult run(BiDiClient client, String claim, CheckOptions options, String context) {
        JsonObject params = new JsonObject();
        ModelSettings.apply(params, options);
        params.addProperty("claim", claim);
        if (options != null && options.record() != null) {
            if (options.record().toString().isEmpty()) throw new IllegalArgumentException("record must be a nonempty path");
            params.addProperty("record", options.record().toAbsolutePath().normalize().toString());
        } else if (context != null) params.addProperty("context", context);
        return new Gson().fromJson(client.send("vibium:check.run", params, 210_000), CheckResult.class);
    }
}
