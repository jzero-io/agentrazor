import MarkdownIt from 'markdown-it';
import hljs from 'highlight.js/lib/common';

const markdown = new MarkdownIt({
  html: false,
  breaks: true,
  linkify: true
});

markdown.renderer.rules.fence = (tokens, index) => {
  const token = tokens[index];
  const info = (token.info || '').trim();
  const lang = info.split(/\s+/)[0]?.toLowerCase() || '';
  const label = lang ? hljs.getLanguage(lang)?.name || lang : 'text';
  let code = '';
  try {
    code = lang && hljs.getLanguage(lang)
      ? hljs.highlight(token.content, { language: lang, ignoreIllegals: true }).value
      : markdown.utils.escapeHtml(token.content);
  } catch {
    code = markdown.utils.escapeHtml(token.content);
  }
  return [
    '<div class="code-block">',
    `<div class="code-block-head"><span class="code-lang">${markdown.utils.escapeHtml(label)}</span></div>`,
    `<pre><code class="hljs${lang ? ` language-${lang}` : ''}">${code}</code></pre>`,
    '</div>'
  ].join('');
};

export function renderMarkdown(content: string) {
  return markdown.render(content);
}
