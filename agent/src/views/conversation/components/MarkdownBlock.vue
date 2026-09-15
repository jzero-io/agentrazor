<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUpdated, ref } from 'vue';
import MarkdownIt from 'markdown-it';
import hljs from 'highlight.js/lib/common';
import { writeClipboardText } from '../../../utils/clipboard';

const props = withDefaults(defineProps<{
  content: string;
  streaming?: boolean;
  normalizeWorkspaceFilePath?: (href: string) => string;
  loadWorkspaceImage?: (path: string) => Promise<Blob>;
}>(), {
  streaming: false,
  normalizeWorkspaceFilePath: undefined,
  loadWorkspaceImage: undefined
});

const emit = defineEmits<{
  openWorkspaceFile: [path: string];
  error: [message: string];
}>();

let mermaidRenderSeq = 0;
let mermaidModulePromise: Promise<typeof import('mermaid')> | undefined;
const workspaceImageUrls = new Map<string, string>();
const workspaceImageLoads = new Map<string, Promise<string>>();
const workspaceImagePlaceholder = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="240" height="135" viewBox="0 0 240 135"%3E%3Crect width="240" height="135" rx="8" fill="%23eef1f2"/%3E%3Ctext x="120" y="72" text-anchor="middle" font-family="sans-serif" font-size="14" fill="%2375818a"%3E%E6%AD%A3%E5%9C%A8%E5%8A%A0%E8%BD%BD%E5%9B%BE%E7%89%87%3C/text%3E%3C/svg%3E';

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

  markdown.renderer.rules.image = (tokens, index, options, _env, renderer) => {
    const src = String(tokens[index].attrGet('src') || '');
    const path = props.normalizeWorkspaceFilePath?.(src) || '';
    if (path) {
      tokens[index].attrSet('src', workspaceImagePlaceholder);
      tokens[index].attrSet('data-workspace-file', 'true');
      tokens[index].attrSet('data-workspace-path', path);
      tokens[index].attrSet('data-workspace-image-state', 'loading');
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
  const link = target?.closest<HTMLElement>('[data-workspace-file]');
  if (!link) return;
  const path = link.dataset.workspacePath
    || props.normalizeWorkspaceFilePath?.(link.getAttribute('href') || link.getAttribute('src') || '')
    || '';
  if (!path) return;
  event.preventDefault();
  event.stopPropagation();
  emit('openWorkspaceFile', path);
}

function workspaceImageUrl(path: string) {
  const existing = workspaceImageUrls.get(path);
  if (existing) return Promise.resolve(existing);
  const pending = workspaceImageLoads.get(path);
  if (pending) return pending;
  if (!props.loadWorkspaceImage) return Promise.reject(new Error('无法加载工作区图片'));
  const load = props.loadWorkspaceImage(path).then(blob => {
    const url = URL.createObjectURL(blob);
    workspaceImageUrls.set(path, url);
    workspaceImageLoads.delete(path);
    return url;
  }).catch(error => {
    workspaceImageLoads.delete(path);
    throw error;
  });
  workspaceImageLoads.set(path, load);
  return load;
}

async function renderWorkspaceImages() {
  const images = Array.from(root.value?.querySelectorAll<HTMLImageElement>('img[data-workspace-path]') || []);
  await Promise.all(images.map(async image => {
    const path = image.dataset.workspacePath || '';
    if (!path || image.dataset.workspaceImageState === 'loaded') return;
    image.dataset.workspaceImageState = 'loading';
    try {
      const url = await workspaceImageUrl(path);
      if (!image.isConnected || image.dataset.workspacePath !== path) return;
      image.src = url;
      image.dataset.workspaceImageState = 'loaded';
    } catch {
      if (image.isConnected) image.dataset.workspaceImageState = 'error';
    }
  }));
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

function scheduleRichContentRender() {
  void nextTick(() => {
    void renderMermaidBlocks();
    void renderWorkspaceImages();
  });
}

onMounted(scheduleRichContentRender);
onUpdated(scheduleRichContentRender);
onBeforeUnmount(() => {
  for (const url of workspaceImageUrls.values()) URL.revokeObjectURL(url);
  workspaceImageUrls.clear();
  workspaceImageLoads.clear();
});
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
