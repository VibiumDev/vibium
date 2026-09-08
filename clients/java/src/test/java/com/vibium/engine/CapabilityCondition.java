package com.vibium.engine;

import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import org.junit.jupiter.api.extension.ConditionEvaluationResult;
import org.junit.jupiter.api.extension.ExecutionCondition;
import org.junit.jupiter.api.extension.ExtensionConfigurationException;
import org.junit.jupiter.api.extension.ExtensionContext;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;

public final class CapabilityCondition implements ExecutionCondition {
    private static final JsonObject MANIFEST = loadManifest();
    private static final String ENGINE = selectedEngine();
    private static final boolean AUDIT = "1".equals(System.getenv("VIBIUM_CAPABILITY_AUDIT"));
    private static final AtomicInteger COLLECTED = new AtomicInteger();
    private static final AtomicInteger SELECTED = new AtomicInteger();
    private static final AtomicInteger SKIPPED = new AtomicInteger();
    private static final Map<String, AtomicInteger> SKIPS = new ConcurrentHashMap<>();
    private static final AtomicBoolean SUMMARY_INSTALLED = new AtomicBoolean();

    static {
        if (SUMMARY_INSTALLED.compareAndSet(false, true)) {
            Runtime.getRuntime().addShutdownHook(new Thread(CapabilityCondition::printSummary));
        }
    }

    @Override
    public ConditionEvaluationResult evaluateExecutionCondition(ExtensionContext context) {
        Set<String> requirements = new LinkedHashSet<>();
        context.getTestClass().map(c -> c.getAnnotation(RequiresCapability.class))
            .ifPresent(marker -> add(requirements, marker));
        context.getTestMethod().map(m -> m.getAnnotation(RequiresCapability.class))
            .ifPresent(marker -> add(requirements, marker));

        for (String name : requirements) {
            if (!MANIFEST.has(name)) {
                throw new ExtensionConfigurationException("unknown capability: " + name);
            }
        }

        // Count concrete test methods, not the container-level class check.
        if (context.getTestMethod().isEmpty()) {
            return ConditionEvaluationResult.enabled("capability class validated");
        }

        COLLECTED.incrementAndGet();
        List<String> missing = new ArrayList<>();
        for (String name : requirements) {
            if (!supports(name, ENGINE)) missing.add(name);
        }
        if (missing.isEmpty()) {
            SELECTED.incrementAndGet();
            return ConditionEvaluationResult.enabled("capabilities supported");
        }

        if (AUDIT && "chrome".equals(ENGINE)) {
            List<String> invalid = new ArrayList<>();
            for (String name : missing) {
                if (MANIFEST.getAsJsonArray(name).size() > 0) invalid.add(name);
            }
            if (!invalid.isEmpty()) {
                throw new ExtensionConfigurationException(
                    "Chrome audit rejected skips for: " + String.join(", ", invalid));
            }
        }

        SKIPPED.incrementAndGet();
        for (String name : missing) {
            SKIPS.computeIfAbsent(name, ignored -> new AtomicInteger()).incrementAndGet();
        }
        return ConditionEvaluationResult.disabled(
            ENGINE + " lacks capabilities: " + String.join(", ", missing));
    }

    private static void add(Set<String> target, RequiresCapability marker) {
        for (String name : marker.value()) target.add(name);
    }

    private static boolean supports(String capability, String engine) {
        JsonArray engines = MANIFEST.getAsJsonArray(capability);
        for (JsonElement candidate : engines) {
            if (engine.equals(candidate.getAsString())) return true;
        }
        return false;
    }

    private static String selectedEngine() {
        String value = System.getenv("VIBIUM_ENGINE");
        String engine = value == null || value.isEmpty() ? "chrome" : value;
        if (!engine.equals("chrome") && !engine.equals("firefox")) {
            throw new ExceptionInInitializerError("unknown VIBIUM_ENGINE: " + engine);
        }
        return engine;
    }

    private static JsonObject loadManifest() {
        String configured = System.getenv("VIBIUM_CAPABILITIES_FILE");
        Path path = configured == null || configured.isEmpty()
            ? Paths.get("../../tests/capabilities.json") : Paths.get(configured);
        try {
            return JsonParser.parseString(Files.readString(path)).getAsJsonObject();
        } catch (IOException | RuntimeException e) {
            throw new ExceptionInInitializerError("cannot read capability manifest " + path + ": " + e);
        }
    }

    private static void printSummary() {
        System.out.printf(
            "capabilities: engine=%s collected=%d selected=%d skipped=%d%n",
            ENGINE, COLLECTED.get(), SELECTED.get(), SKIPPED.get());
        SKIPS.keySet().stream().sorted().forEach(name ->
            System.out.printf("capabilities: skip:%s=%d%n", name, SKIPS.get(name).get()));
    }
}
