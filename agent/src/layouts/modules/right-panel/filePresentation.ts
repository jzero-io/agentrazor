export type FileVisualKind = 'api' | 'code' | 'data' | 'document' | 'image' | 'default';

export function fileVisualKind(name: string, language = ''): FileVisualKind {
  const extension = name.split('.').pop()?.toLowerCase() || '';
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico', 'avif'].includes(extension)) return 'image';
  if (['json', 'jsonc', 'yaml', 'yml', 'toml', 'xml', 'csv'].includes(extension)) return 'data';
  if (['md', 'mdx', 'txt', 'rst', 'pdf'].includes(extension)) return 'document';
  if (['api', 'proto', 'graphql', 'gql'].includes(extension)) return 'api';
  if (
    ['ts', 'tsx', 'js', 'jsx', 'vue', 'go', 'rs', 'py', 'java', 'kt', 'c', 'cc', 'cpp', 'h', 'hpp', 'sh', 'sql', 'css', 'scss', 'html'].includes(extension)
    || ['typescript', 'javascript', 'go', 'rust', 'python', 'java', 'shell', 'sql', 'css', 'html'].includes(language)
  ) return 'code';
  return 'default';
}

export function fileVisualIcon(kind: FileVisualKind) {
  switch (kind) {
    case 'api':
      return 'solar:code-file-linear';
    case 'code':
      return 'solar:file-code-linear';
    case 'data':
      return 'solar:database-linear';
    case 'document':
      return 'solar:document-text-linear';
    case 'image':
      return 'solar:gallery-linear';
    default:
      return 'solar:file-linear';
  }
}

export function fileVisualLabel(kind: FileVisualKind, name: string, language = '') {
  if (kind === 'image') return 'IMG';
  const extension = name.split('.').pop()?.toUpperCase() || '';
  return extension.slice(0, 4) || language.slice(0, 4).toUpperCase() || 'FILE';
}
