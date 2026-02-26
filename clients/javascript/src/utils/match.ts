/**
 * Match a URL against a glob pattern.
 *
 * - `**` matches any characters (including `/`)
 * - `*` matches any characters except `/`
 * - `{a,b,c}` matches any of the alternatives
 * - Other special regex chars are escaped
 * - If no wildcards, performs exact string match
 */
export function matchPattern(pattern: string, url: string): boolean {
  if (!pattern.includes('*') && !pattern.includes('{')) {
    return url === pattern;
  }

  // Extract brace groups before escaping, replace with placeholders
  const braces: string[] = [];
  let withPlaceholders = pattern.replace(/\{([^}]+)\}/g, (_match, inner) => {
    braces.push(inner);
    return `{{BRACE${braces.length - 1}}}`;
  });

  // Escape regex special chars (except *)
  let regex = withPlaceholders.replace(/[.+?^${}()|[\]\\]/g, '\\$&');

  // Replace ** first (before *), then *
  regex = regex.replace(/\*\*/g, '{{GLOBSTAR}}');
  regex = regex.replace(/\*/g, '[^/]*');
  regex = regex.replace(/\{\{GLOBSTAR\}\}/g, '.*');

  // Restore brace groups as regex alternations
  for (let i = 0; i < braces.length; i++) {
    const alternatives = braces[i].split(',').map(s => s.replace(/[.+?^${}()|[\]\\]/g, '\\$&')).join('|');
    regex = regex.replace(`\\{\\{BRACE${i}\\}\\}`, `(?:${alternatives})`);
  }

  return new RegExp(`^${regex}$`).test(url);
}
