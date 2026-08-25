// Dump the JS client's public surface as JSON for the apidrift checker.
//
// The built dist/index.d.ts is the source: TypeScript's private markers only
// exist there (at runtime every member is a plain property), so the .d.ts is
// what actually describes the public API. Keys are the receiver names api.md
// uses in its JS column.
//
//   node scripts/apidrift_js.js | apidrift check -surface js -spec docs/reference/api.md -actual -

const fs = require('fs');
const path = require('path');

const DTS = path.join(__dirname, '..', 'clients', 'javascript', 'dist', 'index.d.ts');

// classMembers parses `declare class X { ... }` blocks into public member
// names. Inline object types (Page's capture namespace) surface as their own
// pseudo-class keyed "X.member".
function parse(dts) {
  const classes = {};
  let current = null; // class name
  let depth = 0;
  let inlineKey = null; // "Page.capture" while inside its object type
  let inlineDepth = 0;

  for (const raw of dts.split('\n')) {
    const line = raw.trim();

    const cls = line.match(/^declare (?:abstract )?class (\w+)/);
    if (cls) {
      current = cls[1];
      classes[current] = classes[current] || new Set();
      depth = 0;
    }
    if (!current) continue;

    if (depth === 1 && !line.startsWith('private') && !line.startsWith('protected')) {
      const member = line.match(/^(?:readonly\s+)?(?:get\s+|set\s+)?([A-Za-z_$][\w$]*)\??\s*[(:<]/);
      if (member && !member[1].startsWith('_') && member[1] !== 'constructor') {
        if (inlineKey === null) {
          classes[current].add(member[1]);
          // An inline object type opens a nested namespace: its members
          // belong to "Class.member", not the class.
          if (/[:{]\s*{\s*$/.test(line)) {
            inlineKey = `${current}.${member[1]}`;
            classes[inlineKey] = classes[inlineKey] || new Set();
            inlineDepth = depth;
          }
        }
      }
    } else if (inlineKey !== null && depth === 2) {
      const member = line.match(/^([A-Za-z_$][\w$]*)\??\s*[(<]/);
      if (member && !member[1].startsWith('_')) {
        classes[inlineKey].add(member[1]);
      }
    }

    for (const ch of line) {
      if (ch === '{') depth++;
      else if (ch === '}') {
        depth--;
        if (inlineKey !== null && depth <= inlineDepth) inlineKey = null;
        if (depth <= 0) current = null;
      }
    }
  }

  const out = {};
  for (const [name, members] of Object.entries(classes)) {
    out[name] = [...members].sort();
  }
  return out;
}

function main() {
  const classes = parse(fs.readFileSync(DTS, 'utf-8'));
  const dist = require(path.join(__dirname, '..', 'clients', 'javascript', 'dist'));

  const surface = {
    // browser is a module-level namespace (start/connect), merged with the
    // Browser instance methods the same rows describe.
    browser: [...new Set([...Object.keys(dist.browser), ...(classes.Browser || [])])].sort(),
    page: classes.Page || [],
    'page.capture': classes['Page.capture'] || [],
    el: classes.Element || [],
    context: classes.BrowserContext || [],
    keyboard: classes.Keyboard || [],
    mouse: classes.Mouse || [],
    touch: classes.Touch || [],
    clock: classes.Clock || [],
    route: classes.Route || [],
    dialog: classes.Dialog || [],
    download: classes.Download || [],
    request: classes.Request || [],
    response: classes.Response || [],
    message: classes.ConsoleMessage || [],
    socket: classes.WebSocketInfo || [],
    recording: classes.Recording || [],
  };

  process.stdout.write(JSON.stringify(surface, null, 1));
}

main();
