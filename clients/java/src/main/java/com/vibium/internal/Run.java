package com.vibium.internal;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.vibium.types.RunResult;
import com.vibium.types.RunOptions;

/** Live semantic operation; all orchestration runs in Go. */
public final class Run {
    private Run() {}
    public static RunResult run(BiDiClient client, String goal, RunOptions options, String context) {
        JsonObject params = new JsonObject();
        ModelSettings.apply(params, options);
        params.addProperty("goal", goal);
        if (context != null) params.addProperty("context", context);
        return new Gson().fromJson(client.send("vibium:run.run", params, 210_000), RunResult.class);
    }
}
