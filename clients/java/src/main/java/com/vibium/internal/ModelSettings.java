package com.vibium.internal;

import com.google.gson.JsonObject;
import com.vibium.types.ModelOptions;

/** Serialize only public overrides onto the existing semantic command. */
final class ModelSettings {
    private ModelSettings() {}
    static void apply(JsonObject params, ModelOptions<?> options) {
        if (options == null) return;
        if (options.provider() != null) params.addProperty("provider", options.provider());
        if (options.model() != null) params.addProperty("model", options.model());
        if (options.aiBaseURL() != null) params.addProperty("aiBaseURL", options.aiBaseURL());
        if (options.reasoningEffort() != null) params.addProperty("reasoningEffort", options.reasoningEffort());
    }
}
