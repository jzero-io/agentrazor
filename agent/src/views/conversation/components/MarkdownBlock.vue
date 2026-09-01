<script setup lang="ts">
import { computed, nextTick, onMounted, onUpdated, ref } from 'vue';
import MarkdownIt from 'markdown-it';
import hljs from 'highlight.js/lib/common';
import { writeClipboardText } from '../../../utils/clipboard';

const props = withDefaults(defineProps<{
  content: string;
  streaming?: boolean;
  normalizeWorkspaceFilePath?: (href: string) => string;
}>(), {
  streaming: false,
  normalizeWorkspaceFilePath: undefined
});

const emit = defineEmits<{
  openWorkspaceFile: [path: string];
  error: [message: string];
}>();

let mermaidRenderSeq = 0;
let mermaidModulePromise: Promise<typeof import('mermaid')> | undefined;

function mermaidCopyButtonHtml() {
  return '<button type="button" class="mermaid-copy-button" data-mermaid-copy>复制源码</button>';
}

function currentMermaidTheme() {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'default';
}

function loadMermaid() {
  if (!mermaidModulePromise) mermaidModulePromise = import('mermaid');
  return mermaidModulePromise;
}

function initMermaid(instance: typeof import('mermaid')['default']) {
  instance.initialize({
    startOnLoad: false,
    securityLevel: 'strict',
    theme: currentMermaidTheme(),
    flowchart: {
      curve: 'basis',
      nodeSpacing: 72,
      rankSpacing: 88,
      padding: 16,
      useMaxWidth: true
    },
    themeVariables: {
      fontSize: '14px'
    }
  });
}

function createMarkdown() {
  const markdown = new MarkdownIt({
    html: false,
    breaks: true,
    linkify: true
  });

  markdown.renderer.rules.link_open = (tokens, index, options, _env, renderer) => {
    const href = String(tokens[index].attrGet('href') || '');
    if (props.normalizeWorkspaceFilePath?.(href)) {
      tokens[index].attrSet('data-workspace-file', 'true');
    } else {
      tokens[index].attrSet('target', '_blank');
      tokens[index].attrSet('rel', 'noopener noreferrer');
    }
    return renderer.renderToken(tokens, index, options);
  };

  markdown.renderer.rules.fence = (tokens, index) => {
    const token = tokens[index];
    const info = (token.info || '').trim();
    const lang = info.split(/\s+/)[0]?.toLowerCase() || '';
    if (lang === 'mermaid') {
      const source = markdown.utils.escapeHtml(token.content);
      return [
        `<div class="mermaid-block" data-mermaid-source="${source}">`,
        mermaidCopyButtonHtml(),
        '<div class="mermaid-placeholder">正在渲染 Mermaid 图表</div>',
        `<pre class="mermaid-source"><code>${source}</code></pre>`,
        '</div>'
      ].join('');
    }
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

  return markdown;
}

const root = ref<HTMLElement | null>(null);
const renderedHtml = computed(() => createMarkdown().render(props.content));

async function copyMermaidSource(event: MouseEvent) {
  const target = event.target instanceof HTMLElement ? event.target : null;
  const button = target?.closest<HTMLButtonElement>('[data-mermaid-copy]');
  if (!button) return;
  const block = button.closest<HTMLElement>('.mermaid-block');
  const source = block?.dataset.mermaidSource || '';
  if (!source) return;
  event.preventDefault();
  event.stopPropagation();
  try {
    await writeClipboardText(source);
    const previous = button.textContent || '复制源码';
    button.textContent = '已复制';
    window.setTimeout(() => {
      if (button.isConnected) button.textContent = previous;
    }, 1200);
  } catch (error) {
    emit('error', error instanceof Error ? error.message : '复制失败');
  }
}

async function handleClick(event: MouseEvent) {
  await copyMermaidSource(event);
  if (event.defaultPrevented) return;
  const target = event.target instanceof HTMLElement ? event.target : null;
  const link = target?.closest<HTMLAnchorElement>('a[data-workspace-file]');
  if (!link) return;
  const path = props.normalizeWorkspaceFilePath?.(link.getAttribute('href') || '') || '';
  if (!path) return;
  event.preventDefault();
  event.stopPropagation();
  emit('openWorkspaceFile', path);
}

async function renderMermaidBlocks() {
  const theme = currentMermaidTheme();
  const blocks = Array.from(root.value?.querySelectorAll<HTMLElement>('.mermaid-block') || []);
  if (!blocks.length) return;
  const { default: mermaid } = await loadMermaid();
  initMermaid(mermaid);
  const markdown = createMarkdown();
  for (const block of blocks) {
    const source = block.dataset.mermaidSource || '';
    if (!source.trim()) continue;
    if (block.dataset.renderedSource === source && block.dataset.renderedTheme === theme) continue;
    block.dataset.renderedSource = source;
    block.dataset.renderedTheme = theme;
    block.classList.remove('is-error');
    block.classList.add('is-rendering');
    try {
      const id = `agent-mermaid-${++mermaidRenderSeq}`;
      const { svg } = await mermaid.render(id, source);
      if (!block.isConnected) continue;
      block.innerHTML = `${mermaidCopyButtonHtml()}<div class="mermaid-diagram">${svg}</div>`;
      block.classList.remove('is-rendering');
    } catch {
      if (!block.isConnected) continue;
      block.classList.remove('is-rendering');
      block.classList.add('is-error');
      block.innerHTML = [
        mermaidCopyButtonHtml(),
        '<div class="mermaid-error">Mermaid 图表渲染失败</div>',
        `<pre class="mermaid-source"><code>${markdown.utils.escapeHtml(source)}</code></pre>`
      ].join('');
    }
  }
}

function scheduleMermaidRender() {
  void nextTick(renderMermaidBlocks);
}

onMounted(scheduleMermaidRender);
onUpdated(scheduleMermaidRender);
</script>

<template>
  <div
    ref="root"
    class="markdown-body"
    :class="{ 'streaming-markdown': streaming }"
    @click="handleClick"
    v-html="renderedHtml"
  />
</template>
