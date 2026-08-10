package com.vibium;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Map;

/**
 * Local HTTP test server for Java tests.
 * Routes match the Python test_server.py so tests are equivalent.
 */
class TestServer {

    private static final String HOME_HTML = "<html><head><title>Test App</title></head><body>"
        + "<h1 class=\"heading\">Welcome to test-app</h1>"
        + "<a href=\"/subpage\">Go to subpage</a>"
        + "<a href=\"/inputs\">Inputs</a>"
        + "<a href=\"/form\">Form</a>"
        + "<p id=\"info\">Some info text</p>"
        + "</body></html>";

    private static final String SUBPAGE_HTML = "<html><head><title>Subpage</title></head><body>"
        + "<h3>Subpage Title</h3>"
        + "<a href=\"/\">Back home</a>"
        + "</body></html>";

    private static final String INPUTS_HTML = "<html><head><title>Inputs</title></head><body>"
        + "<input type=\"text\" id=\"text-input\" />"
        + "<input type=\"number\" id=\"num-input\" />"
        + "<textarea id=\"textarea\"></textarea>"
        + "</body></html>";

    private static final String FORM_HTML = "<html><head><title>Form</title></head><body>"
        + "<form>"
        + "<label for=\"name\">Name</label>"
        + "<input type=\"text\" id=\"name\" name=\"name\" />"
        + "<label for=\"email\">Email</label>"
        + "<input type=\"email\" id=\"email\" name=\"email\" />"
        + "<label for=\"agree\"><input type=\"checkbox\" id=\"agree\" name=\"agree\" /> I agree</label>"
        + "<select id=\"color\" name=\"color\">"
        + "<option value=\"red\">Red</option>"
        + "<option value=\"green\">Green</option>"
        + "<option value=\"blue\">Blue</option>"
        + "</select>"
        + "<button type=\"submit\">Submit</button>"
        + "</form>"
        + "</body></html>";

    private static final String LINKS_HTML = "<html><head><title>Links</title></head><body>"
        + "<ul>"
        + "<li><a href=\"/subpage\" class=\"link\">Link 1</a></li>"
        + "<li><a href=\"/subpage\" class=\"link\">Link 2</a></li>"
        + "<li><a href=\"/subpage\" class=\"link\">Link 3</a></li>"
        + "<li><a href=\"/subpage\" class=\"link special\">Link 4</a></li>"
        + "</ul>"
        + "<div id=\"nested\">"
        + "<span class=\"inner\">Nested span</span>"
        + "<span class=\"inner\">Another span</span>"
        + "</div>"
        + "</body></html>";

    private static final String EVAL_HTML = "<html><head><title>Eval</title></head><body>"
        + "<div id=\"result\"></div>"
        + "<script>window.testVal = 42;</script>"
        + "</body></html>";

    private static final String DIALOG_HTML = "<html><head><title>Dialog</title></head><body>"
        + "<button id=\"alert-btn\" onclick=\"alert('hello')\">Alert</button>"
        + "<button id=\"confirm-btn\" onclick=\"document.getElementById('result').textContent = confirm('sure?')\">Confirm</button>"
        + "<div id=\"result\"></div>"
        + "</body></html>";

    private static final String DYNAMIC_LOADING_HTML = "<html><head><title>Dynamic</title></head><body>"
        + "<div id=\"container\"></div>"
        + "<script>"
        + "setTimeout(() => {"
        + "  const el = document.createElement('div');"
        + "  el.id = 'loaded';"
        + "  el.textContent = 'Loaded!';"
        + "  document.getElementById('container').appendChild(el);"
        + "}, 500);"
        + "</script>"
        + "</body></html>";

    private static final String FETCH_HTML = "<html><head><title>Fetch</title></head><body>"
        + "<div id=\"result\"></div>"
        + "<script>"
        + "async function doFetch() {"
        + "  const res = await fetch('/api/data');"
        + "  const json = await res.json();"
        + "  document.getElementById('result').textContent = JSON.stringify(json);"
        + "}"
        + "</script>"
        + "</body></html>";

    private static final String A11Y_HTML = "<html><head><title>A11y</title></head><body>"
        + "<nav aria-label=\"Main navigation\">"
        + "<a href=\"/subpage\">Go to subpage</a>"
        + "</nav>"
        + "<main>"
        + "<h1>Heading Level 1</h1>"
        + "<button aria-label=\"Close dialog\">X</button>"
        + "<input type=\"checkbox\" id=\"cb\" checked />"
        + "<button disabled>Disabled Button</button>"
        + "</main>"
        // Generic containers carry no role, so they appear only when everything=true.
        + "<div><div><span>plain wrapper text</span></div></div>"
        + "</body></html>";

    private static final String DOWNLOAD_HTML = "<html><head><title>Download</title></head><body>"
        + "<a href=\"/download-file\" id=\"download-link\" download=\"test.txt\">Download</a>"
        + "</body></html>";

    private static final Map<String, String> HTML_ROUTES = new HashMap<>();
    static {
        HTML_ROUTES.put("/", HOME_HTML);
        HTML_ROUTES.put("/subpage", SUBPAGE_HTML);
        HTML_ROUTES.put("/inputs", INPUTS_HTML);
        HTML_ROUTES.put("/form", FORM_HTML);
        HTML_ROUTES.put("/links", LINKS_HTML);
        HTML_ROUTES.put("/eval", EVAL_HTML);
        HTML_ROUTES.put("/dialog", DIALOG_HTML);
        HTML_ROUTES.put("/dynamic-loading", DYNAMIC_LOADING_HTML);
        HTML_ROUTES.put("/fetch", FETCH_HTML);
        HTML_ROUTES.put("/a11y", A11Y_HTML);
        HTML_ROUTES.put("/download", DOWNLOAD_HTML);
    }

    private final HttpServer server;
    private final String baseUrl;

    TestServer() throws IOException {
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        int port = server.getAddress().getPort();
        baseUrl = "http://127.0.0.1:" + port;

        server.createContext("/", exchange -> {
            String path = exchange.getRequestURI().getPath();

            if ("GET".equals(exchange.getRequestMethod())) {
                if ("/api/data".equals(path)) {
                    respond(exchange, 200, "application/json", "{\"message\":\"real data\",\"count\":42}");
                    return;
                }
                if ("/json".equals(path)) {
                    respond(exchange, 200, "application/json", "{\"name\":\"vibium\",\"version\":1}");
                    return;
                }
                if ("/text".equals(path)) {
                    respond(exchange, 200, "text/plain", "hello world");
                    return;
                }
                if ("/download-file".equals(path)) {
                    exchange.getResponseHeaders().set("Content-Type", "application/octet-stream");
                    exchange.getResponseHeaders().set("Content-Disposition", "attachment; filename=\"test.txt\"");
                    byte[] body = "download content".getBytes(StandardCharsets.UTF_8);
                    exchange.sendResponseHeaders(200, body.length);
                    exchange.getResponseBody().write(body);
                    exchange.getResponseBody().close();
                    return;
                }
                if ("/set-cookie".equals(path)) {
                    exchange.getResponseHeaders().add("Set-Cookie", "test_cookie=hello; Path=/");
                    respond(exchange, 200, "text/html", "<html><body>Cookies set!</body></html>");
                    return;
                }

                String html = HTML_ROUTES.getOrDefault(path, HOME_HTML);
                respond(exchange, 200, "text/html", html);
            } else if ("POST".equals(exchange.getRequestMethod())) {
                if ("/api/echo".equals(path)) {
                    byte[] body = exchange.getRequestBody().readAllBytes();
                    respond(exchange, 200, "application/json", "{\"echo\":" + new String(body, StandardCharsets.UTF_8) + "}");
                    return;
                }
                respond(exchange, 404, "text/plain", "Not found");
            }
        });
    }

    void start() {
        server.start();
    }

    void stop() {
        server.stop(0);
    }

    String baseUrl() {
        return baseUrl;
    }

    private static void respond(HttpExchange exchange, int status, String contentType, String body) throws IOException {
        exchange.getResponseHeaders().set("Content-Type", contentType);
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.sendResponseHeaders(status, bytes.length);
        OutputStream os = exchange.getResponseBody();
        os.write(bytes);
        os.close();
    }
}
