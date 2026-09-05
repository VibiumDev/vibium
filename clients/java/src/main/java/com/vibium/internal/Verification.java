package com.vibium.internal;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.vibium.types.VerifyOptions;
import com.vibium.types.VerificationResult;

/** Existing semantic command; provider configuration is read in the Go runtime. */
public final class Verification {
    private Verification() {}
    public static VerificationResult run(BiDiClient client, String claim, VerifyOptions options, String context) {
        JsonObject params = new JsonObject();
        params.addProperty("claim", claim);
        if (options != null && options.record() != null) {
            if (options.record().toString().isEmpty()) throw new IllegalArgumentException("record must be a nonempty path");
            params.addProperty("record", options.record().toAbsolutePath().normalize().toString());
        } else if (context != null) params.addProperty("context", context);
        return new Gson().fromJson(client.send("vibium:verify.run", params, 210_000), VerificationResult.class);
    }
}
