export interface MobileLaunchOptions {
    url?: string; // Default: http://localhost:4723
    capabilities?: Record<string, any>;
}

export class MobileSession {
    private sessionId: string;
    private baseUrl: string;

    constructor(url: string, sessionId: string) {
        this.baseUrl = url.replace(/\/$/, "");
        this.sessionId = sessionId;
    }

    // HTTP Helper
    private async request(method: string, endpoint: string, body?: any): Promise<any> {
        const res = await fetch(`${this.baseUrl}/session/${this.sessionId}${endpoint}`, {
            method,
            headers: { "Content-Type": "application/json" },
            body: body ? JSON.stringify(body) : undefined,
        });
        const json = await res.json();
        if (json.value?.error) throw new Error(json.value.message);
        return json.value;
    }

    async tap(selector: string, strategy = "accessibility id"): Promise<void> {
        const el = await this.request("POST", "/element", { using: strategy, value: selector });
        const elementId = Object.values(el)[0];
        await this.request("POST", `/element/${elementId}/click`, {});
    }

    async type(selector: string, text: string, strategy = "accessibility id"): Promise<void> {
        const el = await this.request("POST", "/element", { using: strategy, value: selector });
        const elementId = Object.values(el)[0];
        await this.request("POST", `/element/${elementId}/value`, { text });
    }

    /**
     * Wait for an element to exist.
     * @param selector Selector to find
     * @param timeout Timeout in milliseconds (default: 10000)
     * @param strategy Locator strategy (default: "accessibility id")
     */
    async waitFor(selector: string, timeout = 10000, strategy = "accessibility id"): Promise<void> {
        const start = Date.now();
        while (Date.now() - start < timeout) {
            try {
                // Try to find the element
                await this.request("POST", "/element", { using: strategy, value: selector });
                return; // Found!
            } catch (err) {
                // Not found yet (or other error), wait and retry
                await new Promise(resolve => setTimeout(resolve, 500));
            }
        }
        throw new Error(`Timeout waiting for element '${selector}' after ${timeout}ms`);
    }

    async source(): Promise<string> {
        return this.request("GET", "/source");
    }

    async setGeoLocation(location: { latitude: number; longitude: number; altitude?: number }): Promise<void> {
        await this.request("POST", "/location", { location });
    }

    async tapCoordinates(x: number, y: number): Promise<void> {
        const actions = {
            actions: [{
                type: "pointer",
                id: "finger1",
                parameters: { pointerType: "touch" },
                actions: [
                    { type: "pointerMove", duration: 0, x, y },
                    { type: "pointerDown", button: 0 },
                    { type: "pause", duration: 200 },
                    { type: "pointerUp", button: 0 }
                ]
            }]
        };
        await this.request("POST", "/actions", actions);
    }

    async quit(): Promise<void> {
        await fetch(`${this.baseUrl}/session/${this.sessionId}`, { method: "DELETE" });
    }
}

export const mobile = {
    async connect(options: MobileLaunchOptions = {}): Promise<MobileSession> {
        const url = options.url || "http://localhost:4723";
        const capabilities = options.capabilities || {};

        // Start session
        const res = await fetch(`${url}/session`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                capabilities: {
                    alwaysMatch: capabilities
                }
            }),
        });
        const json = await res.json();

        // Check for error in response
        if (json.value?.error) {
            throw new Error(`Failed to create session: ${json.value.message} (${json.value.error})`);
        }

        const sessionId = json.value.sessionId;

        return new MobileSession(url, sessionId);
    }
};
