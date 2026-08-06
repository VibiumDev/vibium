// Provider adapters for the cloud benchmark.
//
// Each provider declares the env vars that activate it, how to obtain a
// connectable URL (a static classic/BiDi URL, or a mint() REST call), and
// how to clean up. Grid providers need no mint(): vibium's classic
// WebDriver support creates and deletes the session itself.
//
// CDP-only providers (Browserbase, Browser-Use, Hyperbrowser) carry
// mintOnly: vibium speaks WebDriver BiDi, not CDP, so until the CDP→BiDi
// shim exists the bench can only time their session mint/delete REST calls.

const env = (k) => process.env[k];

async function jsonFetch(url, options) {
  const res = await fetch(url, options);
  const body = await res.text();
  if (!res.ok) throw new Error(`${options?.method || 'GET'} ${url} → ${res.status}: ${body.slice(0, 300)}`);
  return body ? JSON.parse(body) : null;
}

const PROVIDERS = [
  {
    // Control run: local browser launch through the same client code path.
    name: 'local',
    available: () => true,
    missing: () => '',
    url: undefined,
  },
  {
    // Second control: any classic WebDriver endpoint you point it at —
    // a local chromedriver, dockerized Selenium, or a DIY box through an
    // SSH tunnel / fly proxy (see deploy kits). Exercises the exact code
    // path the cloud grids use.
    name: 'diy-tunnel',
    available: () => !!env('DIY_TUNNEL_URL'),
    missing: () => 'DIY_TUNNEL_URL (e.g. http://127.0.0.1:9515 with a tunnel up)',
    // DIY_TUNNEL_NAME tags results with which box was behind the tunnel
    // (hetzner, flyio, macmini, ...) — selection stays --provider diy-tunnel.
    get resultName() {
      const label = env('DIY_TUNNEL_NAME');
      return label ? `diy-${label}` : 'diy-tunnel';
    },
    get url() { return env('DIY_TUNNEL_URL'); },
    caps: { 'goog:chromeOptions': { args: ['--headless=new'] } },
  },
  {
    name: 'kernel',
    available: () => !!env('KERNEL_API_KEY'),
    missing: () => 'KERNEL_API_KEY',
    mint: async () => {
      const b = await jsonFetch('https://api.onkernel.com/browsers', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${env('KERNEL_API_KEY')}`,
          'Content-Type': 'application/json',
        },
        // headless: the API default is headful, which bills at 8x.
        body: JSON.stringify({ headless: true, timeout_seconds: 300 }),
      });
      return {
        url: b.webdriver_ws_url, // BiDi, JWT rides in the query string
        cleanup: () => jsonFetch(`https://api.onkernel.com/browsers/${b.session_id}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${env('KERNEL_API_KEY')}` },
        }).catch(() => {}),
      };
    },
  },
  {
    name: 'browserstack',
    available: () => !!(env('BROWSERSTACK_USERNAME') && env('BROWSERSTACK_ACCESS_KEY')),
    missing: () => 'BROWSERSTACK_USERNAME + BROWSERSTACK_ACCESS_KEY',
    get url() {
      return `https://${env('BROWSERSTACK_USERNAME')}:${env('BROWSERSTACK_ACCESS_KEY')}@hub-cloud.browserstack.com/wd/hub`;
    },
    caps: {
      browserName: 'chrome',
      'bstack:options': {
        seleniumVersion: '4.20.0',
        seleniumBidi: 'true',
        sessionName: 'vibium-bench',
        idleTimeout: 300,
      },
    },
  },
  {
    name: 'saucelabs',
    available: () => !!(env('SAUCE_USERNAME') && env('SAUCE_ACCESS_KEY')),
    missing: () => 'SAUCE_USERNAME + SAUCE_ACCESS_KEY',
    get url() {
      const region = env('SAUCE_REGION') || 'us-west-1';
      return `https://${env('SAUCE_USERNAME')}:${env('SAUCE_ACCESS_KEY')}@ondemand.${region}.saucelabs.com/wd/hub`;
    },
    // Note: Sauce docs cap CDP/BiDi sessions at 10 minutes.
    caps: {
      browserName: 'chrome',
      'sauce:options': { name: 'vibium-bench', idleTimeout: 300 },
    },
  },
  {
    name: 'lambdatest',
    available: () => !!(env('LT_USERNAME') && env('LT_ACCESS_KEY')),
    missing: () => 'LT_USERNAME + LT_ACCESS_KEY',
    get url() {
      return `https://${env('LT_USERNAME')}:${env('LT_ACCESS_KEY')}@hub.lambdatest.com/wd/hub`;
    },
    caps: {
      browserName: 'chrome',
      browserVersion: 'latest',
      'LT:Options': { name: 'vibium-bench', idleTimeout: 300 },
    },
  },
  {
    name: 'testingbot',
    available: () => !!(env('TESTINGBOT_KEY') && env('TESTINGBOT_SECRET')),
    missing: () => 'TESTINGBOT_KEY + TESTINGBOT_SECRET',
    get url() {
      return `https://${env('TESTINGBOT_KEY')}:${env('TESTINGBOT_SECRET')}@hub.testingbot.com/wd/hub`;
    },
    // Their BiDi doc says to pin a selenium-version in tb:options; the
    // accepted values are unverified until the first live run.
    caps: {
      browserName: 'chrome',
      'tb:options': { name: 'vibium-bench' },
    },
  },
  {
    name: 'browserbase',
    available: () => !!env('BROWSERBASE_API_KEY'),
    missing: () => 'BROWSERBASE_API_KEY',
    mintOnly: true, // exposes CDP + classic Selenium only; no BiDi
    mint: async () => {
      const s = await jsonFetch('https://api.browserbase.com/v1/sessions', {
        method: 'POST',
        headers: {
          'X-BB-API-Key': env('BROWSERBASE_API_KEY'),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      });
      return {
        url: s.connectUrl, // CDP — vibium cannot drive this yet
        cleanup: () => jsonFetch(`https://api.browserbase.com/v1/sessions/${s.id}`, {
          method: 'POST',
          headers: {
            'X-BB-API-Key': env('BROWSERBASE_API_KEY'),
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ status: 'REQUEST_RELEASE' }),
        }).catch(() => {}),
      };
    },
  },
  {
    name: 'browser-use',
    available: () => !!env('BROWSER_USE_API_KEY'),
    missing: () => 'BROWSER_USE_API_KEY',
    mintOnly: true, // CDP only
    mint: async () => {
      const b = await jsonFetch('https://api.browser-use.com/api/v4/browsers', {
        method: 'POST',
        headers: {
          'X-Browser-Use-API-Key': env('BROWSER_USE_API_KEY'),
          'Content-Type': 'application/json',
        },
        // default proxyCountryCode 'us' turns on a $5/GB residential proxy
        body: JSON.stringify({ proxyCountryCode: null }),
      });
      return {
        url: b.cdpUrl, // CDP — vibium cannot drive this yet
        cleanup: () => jsonFetch(`https://api.browser-use.com/api/v4/browsers/${b.id}`, {
          method: 'PATCH',
          headers: {
            'X-Browser-Use-API-Key': env('BROWSER_USE_API_KEY'),
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ action: 'stop' }),
        }).catch(() => {}),
      };
    },
  },
  {
    name: 'cloudflare',
    available: () => !!(env('CLOUDFLARE_ACCOUNT_ID') && env('CLOUDFLARE_API_TOKEN')),
    missing: () => 'CLOUDFLARE_ACCOUNT_ID + CLOUDFLARE_API_TOKEN',
    mintOnly: true, // Browser Run is CDP only; no BiDi
    mint: async () => {
      const base = `https://api.cloudflare.com/client/v4/accounts/${env('CLOUDFLARE_ACCOUNT_ID')}/browser-rendering/devtools/browser`;
      const headers = { Authorization: `Bearer ${env('CLOUDFLARE_API_TOKEN')}` };
      const s = await jsonFetch(base, {
        method: 'POST',
        headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      const id = s?.result?.session_id ?? s?.session_id;
      return {
        url: null, // CDP — vibium cannot drive this yet
        cleanup: () => (id
          ? jsonFetch(`${base}/${id}`, { method: 'DELETE', headers }).catch(() => {})
          : Promise.resolve()),
      };
    },
  },
  {
    name: 'hyperbrowser',
    available: () => !!env('HYPERBROWSER_API_KEY'),
    missing: () => 'HYPERBROWSER_API_KEY',
    mintOnly: true, // CDP only
    mint: async () => {
      const s = await jsonFetch('https://api.hyperbrowser.ai/api/session', {
        method: 'POST',
        headers: {
          'x-api-key': env('HYPERBROWSER_API_KEY'),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      });
      return {
        url: s.wsEndpoint, // CDP — vibium cannot drive this yet
        cleanup: () => jsonFetch(`https://api.hyperbrowser.ai/api/session/${s.id}/stop`, {
          method: 'PUT',
          headers: { 'x-api-key': env('HYPERBROWSER_API_KEY') },
        }).catch(() => {}),
      };
    },
  },
];

module.exports = { PROVIDERS };
