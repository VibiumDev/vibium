package com.vibium;

import com.vibium.types.A11yNode;
import com.vibium.types.A11yOptions;
import com.vibium.types.StartOptions;
import com.vibium.types.WaitOptions;
import org.junit.jupiter.api.*;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Covers commands where the Java client sent a param name the engine does not read.
 * Each of these silently no-opped or failed before, because the engine reads params
 * out of an untyped map and a key it does not recognise simply becomes a zero value.
 */
class WireContractTest {

    static Browser browser;
    static TestServer server;
    Page page;

    @BeforeAll
    static void setup() throws Exception {
        server = new TestServer();
        server.start();
        browser = Vibium.start(new StartOptions().headless(true));
    }

    @AfterAll
    static void teardown() {
        if (browser != null) browser.stop();
        if (server != null) server.stop();
    }

    @BeforeEach
    void beforeEach() {
        page = browser.page();
    }

    /**
     * The client sent "expression"; the engine reads "fn". fn was therefore "",
     * and the engine's `const __vibiumPred = (%s)` became `()` — the
     * "SyntaxError: Unexpected token ')'" in issue #174.
     */
    @Test
    void waitForFunctionEvaluatesTheExpression() {
        page.go(server.baseUrl());
        Object result = page.waitForFunction(
            "() => document.querySelector('h1') !== null",
            new WaitOptions().timeout(5000));
        assertNotNull(result, "waitForFunction should resolve with the truthy value");
    }

    @Test
    void waitForFunctionAcceptsABareExpression() {
        page.go(server.baseUrl());
        Object result = page.waitForFunction(
            "document.readyState === 'complete'",
            new WaitOptions().timeout(5000));
        assertNotNull(result, "a bare expression should be wrapped and evaluated");
    }

    /**
     * The client sent the parent under "selector" plus a "childSelector" the engine
     * never reads, so the lookup was never scoped to this element.
     */
    @Test
    void findIsScopedToTheParentElement() {
        page.go(server.baseUrl() + "/links");
        Element nested = page.find("#nested");

        // info() is what the engine resolved under the scope, so this asserts the
        // scoping itself rather than a second, unscoped lookup.
        Element inner = nested.find("span");
        assertEquals("span", inner.info().tag());
        assertEquals("Nested span", inner.info().text());
    }

    @Test
    void findAllIsScopedToTheParentElement() {
        page.go(server.baseUrl() + "/links");

        List<Element> allLinks = page.findAll("a");
        assertEquals(4, allLinks.size(), "page has four links in total");

        List<Element> nestedSpans = page.find("#nested").findAll("span");
        assertEquals(2, nestedSpans.size(), "only the two spans inside #nested are in scope");
    }

    /**
     * The client sent "interestingOnly"; the engine reads "everything" and inverts it,
     * so the option never had any effect.
     */
    @Test
    void a11yEverythingReturnsMoreNodesThanTheDefault() {
        page.go(server.baseUrl() + "/a11y");

        int interesting = countNodes(page.a11yTree());
        int everything = countNodes(page.a11yTree(new A11yOptions().everything(true)));

        assertTrue(everything > interesting,
            "everything(true) should include uninteresting nodes: "
                + everything + " vs " + interesting);
    }

    private static int countNodes(A11yNode node) {
        if (node == null) return 0;
        int total = 1;
        if (node.children() != null) {
            for (A11yNode child : node.children()) {
                total += countNodes(child);
            }
        }
        return total;
    }
}
