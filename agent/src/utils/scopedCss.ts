export function scopeCssSelectors(css: string, scope: string): string {
  const rules: string[] = [];
  let buffer = '';
  for (const line of css.split('\n')) {
    buffer += `${line}\n`;
    if (!line.includes('}')) continue;
    const rule = buffer;
    buffer = '';
    const openBrace = rule.indexOf('{');
    if (openBrace < 0) {
      rules.push(rule);
      continue;
    }
    const selectors = rule.slice(0, openBrace);
    const body = rule.slice(openBrace);
    const scoped = selectors
      .split(',')
      .map(selector => `${scope} ${selector.trim()}`)
      .join(', ');
    rules.push(`${scoped}${body}`);
  }
  if (buffer.trim()) rules.push(buffer);
  return rules.join('');
}

export function installScopedCss(css: string, scope: string) {
  const style = document.createElement('style');
  style.textContent = scopeCssSelectors(css, scope);
  document.head.appendChild(style);
  return style;
}
