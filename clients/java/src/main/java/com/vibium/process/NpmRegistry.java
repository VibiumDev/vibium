package com.vibium.process;

import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.vibium.VibiumException;
import com.vibium.bidi.BiDiClient;

import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

final class NpmRegistry {
    private NpmRegistry() {}

    static String resolveTarballUrl(String packageName, String requestedVersionOrTag) {
        requireNonBlank(packageName, "packageName");
        requireNonBlank(requestedVersionOrTag, "requestedVersionOrTag");

        String encoded = URLEncoder.encode(packageName, StandardCharsets.UTF_8);
        String url = "https://registry.npmjs.org/" + encoded;

        JsonObject packument = fetchJson(url);
        if (!packument.has("versions") || !packument.get("versions").isJsonObject()) {
            throw new VibiumException("npm registry response missing versions for " + packageName);
        }

        String resolvedVersion = requestedVersionOrTag;
        JsonObject versions = packument.getAsJsonObject("versions");
        if (!versions.has(resolvedVersion)) {
            // Handle dist-tags like "latest" (and fallback if exact version not found).
            if (packument.has("dist-tags") && packument.get("dist-tags").isJsonObject()) {
                JsonObject tags = packument.getAsJsonObject("dist-tags");
                if (tags.has(requestedVersionOrTag)) {
                    resolvedVersion = tags.get(requestedVersionOrTag).getAsString();
                } else if (tags.has("latest")) {
                    resolvedVersion = tags.get("latest").getAsString();
                }
            }
        }

        if (!versions.has(resolvedVersion) || !versions.get(resolvedVersion).isJsonObject()) {
            throw new VibiumException("npm package " + packageName + " does not have version " + resolvedVersion);
        }

        JsonObject v = versions.getAsJsonObject(resolvedVersion);
        JsonObject dist = v.has("dist") && v.get("dist").isJsonObject() ? v.getAsJsonObject("dist") : null;
        if (dist == null || !dist.has("tarball")) {
            throw new VibiumException("npm package " + packageName + " version " + resolvedVersion + " missing dist.tarball");
        }
        return dist.get("tarball").getAsString();
    }

    private static JsonObject fetchJson(String url) {
        try {
            HttpClient client = HttpClient.newBuilder()
                    .connectTimeout(Duration.ofSeconds(20))
                    .followRedirects(HttpClient.Redirect.NORMAL)
                    .build();
            HttpRequest req = HttpRequest.newBuilder()
                    .uri(URI.create(url))
                    .timeout(Duration.ofSeconds(30))
                    .header("Accept", "application/json")
                    .header("User-Agent", "vibium-java")
                    .GET()
                    .build();

            HttpResponse<String> resp = client.send(req, HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8));
            if (resp.statusCode() != 200) {
                throw new VibiumException("npm registry request failed: " + resp.statusCode() + " for " + url);
            }

            JsonElement el = BiDiClient.GSON.fromJson(resp.body(), JsonElement.class);
            if (el == null || !el.isJsonObject()) {
                throw new VibiumException("Invalid npm registry JSON for " + url);
            }
            return el.getAsJsonObject();
        } catch (VibiumException e) {
            throw e;
        } catch (Exception e) {
            throw new VibiumException("Failed to query npm registry: " + url, e);
        }
    }

    private static void requireNonBlank(String value, String name) {
        if (value == null || value.isBlank()) {
            throw new IllegalArgumentException(name + " must be non-blank");
        }
    }
}
