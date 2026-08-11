import type { ThreadItem, Turn } from './api';

export type DisplayThreadItem = ThreadItem & {
  memberIds?: string[];
  searchQueries?: string[];
  skillName?: string;
  skillFiles?: string[];
};

export interface ProcessDisplayItem {
  item: DisplayThreadItem;
  label: string;
  icon: string;
  detail: string;
  live: boolean;
  expandable: boolean;
  className: string;
}

type ThreadCommandAction = NonNullable<ThreadItem['commandActions']>[number];
type PathCommandAction = Extract<ThreadCommandAction, { path: string | null }>;

export function formatTurnDuration(durationMs = 0) {
  const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts: string[] = [];
  if (hours) parts.push(`${hours}小时`);
  if (minutes) parts.push(`${minutes}分`);
  if (seconds || parts.length === 0) parts.push(`${seconds}秒`);
  return parts.join('');
}

export function isStoppedTurn(turn: Turn) {
  return String(turn.status || '').replace(/[-_\s]/g, '').toLowerCase() === 'stopped';
}

export function completedProcessSummary(turn: Turn) {
  const duration = formatTurnDuration(turn.durationMs ?? 0);
  return isStoppedTurn(turn) ? `你在 ${duration} 后停止了` : `已处理 ${duration}`;
}

export function streamingProcessSummary(_turn: Turn, elapsedDurationMs: number) {
  return `正在处理 ${formatTurnDuration(elapsedDurationMs)}`;
}

export function showCompletedProcessSummary(turn: Turn) {
  return turn.durationMs !== undefined && (isStoppedTurn(turn) || turnProcessItems(turn).length > 0);
}

export function humanizeSkillName(name: string): string {
  return name
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function turnProcessItems(turn: Turn) {
  return turn.items.filter(item =>
    item.type !== 'userMessage'
    && item.type !== 'imageGeneration'
    && item.type !== 'reasoning'
    && (item.type !== 'agentMessage' || isIntermediateAgentMessage(item))
  );
}

export function turnWorkedItems(turn: Turn) {
  return turnProcessItems(turn).filter(item => item.type !== 'agentMessage');
}

export function turnResultItems(turn: Turn) {
  return turn.items.filter(item =>
    item.type === 'imageGeneration'
    || item.type === 'agentMessage' && !isIntermediateAgentMessage(item)
  );
}

export function turnActiveProcessItem(turn: Turn): ThreadItem | null {
  const items = turnProcessItems(turn).filter(shouldDisplayProcessItem);
  for (let index = items.length - 1; index >= 0; index -= 1) {
    if (isLiveProcessItem(items[index])) return items[index];
  }
  return null;
}

export function isLiveProcessItem(item: ThreadItem) {
  const streamStatus = normalizedStatus(item.streamStatus);
  if (streamStatus === 'running') return true;
  if (streamStatus === 'completed') return false;

  const status = normalizedStatus(item.status);
  if (['inprogress', 'running', 'pending', 'started'].includes(status)) return true;
  if (['completed', 'failed', 'cancelled', 'canceled', 'interrupted', 'succeeded', 'success', 'done'].includes(status)) return false;

  if (item.type === 'commandExecution') {
    return item.exitCode === undefined && item.durationMs === undefined && item.aggregatedOutput === undefined;
  }
  return false;
}

export function activityTitle(item: ThreadItem, live = false) {
  switch (item.type) {
    case 'commandExecution': return commandExecutionTitle(item, live);
    case 'fileChange': return fileChangeTitle(item, live);
    case 'webSearch': return webSearchTitle(item, live);
    case 'imageView': return live ? '正在查看图片' : '查看图片';
    case 'imageGeneration': return live ? '正在生成图片' : '生成图片';
    case 'mcpToolCall': return `${live ? '正在调用' : '调用'} ${item.server || 'MCP'} · ${item.tool || '工具'}`;
    case 'dynamicToolCall': return `${live ? '正在调用' : '调用'} ${item.tool || '工具'}`;
    case 'collabAgentToolCall': return live ? '正在协作处理' : '协作处理';
    case 'contextCompaction': return live ? '正在整理上下文' : '整理上下文';
    case 'plan': return '计划';
    default: return live ? '正在处理' : String(item.type);
  }
}

export function activityIcon(item: ThreadItem) {
  switch (item.type) {
    case 'commandExecution': return isSkillCommandItem(item) ? 'solar:document-text-linear' : 'solar:command-linear';
    case 'fileChange': return 'solar:document-add-linear';
    case 'webSearch': return isLocalFileOpenItem(item) ? 'solar:document-text-linear' : webSearchActionType(item) === 'openPage' ? 'solar:link-round-angle-linear' : 'solar:magnifer-linear';
    case 'imageGeneration': return 'solar:gallery-add-linear';
    case 'reasoning': return 'solar:lightbulb-bolt-linear';
    default: return 'solar:widget-5-linear';
  }
}

export function activityDetail(item: ThreadItem) {
  if (item.type === 'commandExecution') return commandExecutionDetail(item);
  if (item.type === 'plan') return item.text || '';
  if (item.type === 'fileChange') return fileChangeDetail(item);
  if (item.type === 'webSearch') return item.result != null ? String(item.result) : '';
  if (item.type === 'mcpToolCall' || item.type === 'dynamicToolCall') {
    return JSON.stringify({ arguments: item.arguments, result: item.result }, null, 2);
  }
  return '';
}

export function processDisplayItems(turn: Turn, streaming = false): ProcessDisplayItem[] {
  const active = streaming ? turnActiveProcessItem(turn) : null;
  const visibleItems = turnDisplayItems(turn).filter(item => item.type !== 'reasoning') as DisplayThreadItem[];
  const items = compactProcessItemsByStage(visibleItems);
  return items.map(item => {
    const live = Boolean(active && (item.id === active.id || item.memberIds?.includes(active.id)));
    const skillGroup = item.type === 'skillReadGroup';
    const skillRead = isSkillCommandItem(item);
    const label = item.type === 'webSearchGroup'
      ? (live ? '正在搜索网页' : '已搜索网页')
      : skillGroup
        ? `${live ? '正在' : '已'}读取 ${item.skillName || 'Skills'} 技能`
        : activityTitle(item, live);
    const detail = item.type === 'webSearchGroup' || skillGroup || skillRead ? '' : activityDetail(item);
    return {
      item,
      label,
      icon: item.type === 'webSearchGroup' ? 'solar:global-linear' : skillGroup ? 'solar:document-text-linear' : activityIcon(item),
      detail,
      live,
      expandable: Boolean(detail) || skillGroup && Boolean(item.skillFiles?.length),
      className: processClassName(item)
    };
  });
}

function normalizedStatus(value: unknown) {
  return typeof value === 'string' ? value.replace(/[-_\s]/g, '').toLowerCase() : '';
}

function isIntermediateAgentMessage(item: ThreadItem) {
  return item.type === 'agentMessage' && item.phase === 'commentary';
}

function webSearchActionType(item: ThreadItem) {
  return (item.action as { type?: string } | undefined)?.type || '';
}

function cleanToolTraceSuffix(value: string) {
  return value.replace(/#ws_call_id=.*$/, '').trim();
}

function cleanSearchQuery(query: string): string {
  return cleanToolTraceSuffix(query).replace(/\s*(?:\.\.\.|…)+\s*$/, '').trim();
}

function webSearchUrl(item: ThreadItem) {
  const action = item.action as Record<string, unknown> | undefined;
  const values = [action?.url, item.url].filter(value => typeof value === 'string' && value.trim()) as string[];
  return values.length ? cleanToolTraceSuffix(values[0]) : '';
}

function isLocalFileOpenItem(item: ThreadItem) {
  return webSearchActionType(item) === 'openPage' && webSearchUrl(item).startsWith('file://');
}

function localFileDisplayName(url: string) {
  const path = url.replace(/^file:\/\//, '');
  const skillMatch = path.match(/\/skills\/([^/]+)(?:\/(.*))?$/);
  if (skillMatch) {
    const skill = humanizeSkillName(skillMatch[1]);
    const file = skillMatch[2]?.split('/').filter(Boolean).pop();
    return file ? `${skill} 技能：${file}` : `${skill} 技能`;
  }
  return path.split('/').filter(Boolean).pop() || path;
}

function webSearchText(item: ThreadItem) {
  const action = item.action as Record<string, unknown> | undefined;
  const values = [
    action?.query,
    item.query,
    action?.title,
    action?.url
  ].filter(value => typeof value === 'string' && value.trim()) as string[];
  return values.length ? cleanSearchQuery(values[0]) : '';
}

function webSearchTitle(item: ThreadItem, live = false) {
  if (isLocalFileOpenItem(item)) {
    const prefix = live ? '正在读取文件' : '读取文件';
    const text = localFileDisplayName(webSearchUrl(item));
    return text ? `${prefix}：${text}` : prefix;
  }
  const openPage = webSearchActionType(item) === 'openPage';
  const prefix = live
    ? openPage ? '正在打开网页' : '正在搜索网页'
    : openPage ? '打开网页' : '搜索网页';
  const text = webSearchText(item);
  return text ? `${prefix}：${text}` : prefix;
}

function fileChangeTitle(item: ThreadItem, live = false) {
  const changes = Array.isArray(item.changes) ? item.changes as Array<Record<string, unknown>> : [];
  const prefix = live ? '正在修改文件' : '修改文件';
  return changes.length ? `${prefix}：${changes.length} 个` : prefix;
}

function fileChangeDetail(item: ThreadItem) {
  const changes = Array.isArray(item.changes) ? item.changes as Array<Record<string, unknown>> : [];
  return changes.map(change => String(change.path || '')).filter(Boolean).join('\n');
}

function shellInnerCommand(command: string) {
  const trimmed = command.trim();
  const match = trimmed.match(/^\/bin\/sh\s+-lc\s+(['"])([\s\S]*)\1$/);
  return match ? match[2].trim() : trimmed;
}

function commandFromActions(item: ThreadItem) {
  if (!Array.isArray(item.commandActions)) return '';
  return item.commandActions
    .map(action => action.command.trim())
    .find(Boolean) || '';
}

function commandCandidate(item: ThreadItem) {
  if (typeof item.command === 'string' && item.command.trim()) return item.command.trim();
  return commandFromActions(item);
}

function truncateCommandTitle(command: string) {
  const value = shellInnerCommand(command).replace(/\s+/g, ' ').trim();
  return value.length > 96 ? `${value.slice(0, 96)}...` : value;
}

function cleanCommandOutput(output: string) {
  return output
    .replace(/\x1B\[[0-?]*[ -/]*[@-~]/g, '')
    .replace(/(^|[^\[])(?:\[(?:\d{1,3};)*\d{1,3}m)+/g, '$1')
    .trimEnd();
}

function commandExecutionTitle(item: ThreadItem, live = false) {
  if (isSkillCommandItem(item)) return skillCommandTitle(item, live);
  const command = commandCandidate(item);
  return command
    ? `${live ? '正在' : '已'}执行命令：${truncateCommandTitle(command)}`
    : `${live ? '正在' : '已'}执行命令`;
}

function commandExecutionDetail(item: ThreadItem) {
  const command = commandCandidate(item) ? shellInnerCommand(commandCandidate(item)) : '';
  const output = typeof item.aggregatedOutput === 'string' ? cleanCommandOutput(item.aggregatedOutput) : '';
  return [command, output].filter(Boolean).join('\n\n');
}

function commandActionPath(action: ThreadCommandAction): string {
  if (!('path' in action) || typeof action.path !== 'string') return '';
  return action.path;
}

function isSkillAction(action: ThreadCommandAction): action is PathCommandAction {
  return commandActionPath(action).includes('/skills/');
}

function commandSkillPath(item: ThreadItem): string {
  const command = shellInnerCommand(commandCandidate(item));
  const match = command.match(/(?:^|\s)(?:cat|find|ls|sed\s+-n\s+[^\s]+)\s+['"]?([^'"\s|;&]*\/skills\/+[^'"\s|;&]+)/);
  return match ? match[1] : '';
}

function isSkillCommandItem(item: ThreadItem): boolean {
  if (item.type !== 'commandExecution') return false;
  if (Array.isArray(item.commandActions) && item.commandActions.some(isSkillAction)) return true;
  return Boolean(commandSkillPath(item));
}

function skillAction(item: ThreadItem): PathCommandAction | undefined {
  if (!Array.isArray(item.commandActions)) return undefined;
  return item.commandActions.find(isSkillAction);
}

function skillActionPath(item: ThreadItem): string {
  const action = skillAction(item);
  return action ? commandActionPath(action) : commandSkillPath(item);
}

function skillPathParts(item: ThreadItem): { skill: string; rest: string } {
  const match = skillActionPath(item).match(/\/skills\/+([^/]+)(?:\/(.*))?$/);
  return { skill: match?.[1] || '', rest: match?.[2] || '' };
}

function skillReadName(item: ThreadItem): string {
  return skillPathParts(item).skill;
}

function skillReadFileName(item: ThreadItem): string {
  const rest = skillPathParts(item).rest;
  return rest ? rest.split('/').filter(Boolean).pop() ?? '' : '';
}

function skillCommandTitle(item: ThreadItem, live = false) {
  const action = skillAction(item);
  const command = shellInnerCommand(commandCandidate(item));
  const skill = skillReadName(item);
  const skillLabel = skill ? humanizeSkillName(skill) : '';
  const prefix = live ? '正在' : '已';
  if (action?.type === 'listFiles' || /^(?:find|ls)\s+/.test(command)) return `${prefix}列${skillLabel ? ` ${skillLabel} 技能` : '技能'}文件`;
  return `${prefix}读取${skillLabel ? ` ${skillLabel} 技能` : '技能'}`;
}

function shouldDisplayProcessItem(item: ThreadItem) {
  return item.type !== 'webSearch' || webSearchActionType(item) !== 'openPage';
}

function createSkillReadGroup(items: ThreadItem[]): DisplayThreadItem | null {
  if (!items.length) return null;
  const first = items[0];
  const skill = skillReadName(first);
  const files = [...new Set(items.map(skillReadFileName).filter(Boolean))];
  return {
    ...first,
    id: `skill-read-group-${first.id}-${items.length}`,
    type: 'skillReadGroup',
    memberIds: items.map(item => item.id),
    skillName: skill ? humanizeSkillName(skill) : 'Skills',
    skillFiles: files
  } as DisplayThreadItem;
}

function turnDisplayItems(turn: Turn): ThreadItem[] {
  const result: DisplayThreadItem[] = [];
  let skillItems: ThreadItem[] = [];
  const flushSkills = () => {
    const group = createSkillReadGroup(skillItems);
    if (group) result.push(group);
    skillItems = [];
  };

  for (const item of turnProcessItems(turn).filter(shouldDisplayProcessItem)) {
    if (isSkillCommandItem(item)) {
      skillItems.push(item);
      continue;
    }
    flushSkills();
    result.push(item);
  }
  flushSkills();
  return result;
}

function processClassName(item: DisplayThreadItem) {
  if (item.type === 'webSearchGroup') return 'web-search-group-entry';
  if (item.type === 'skillReadGroup') return 'skill-read-entry';
  if (item.type === 'webSearch') return 'search-entry';
  if (item.type === 'commandExecution') return isSkillCommandItem(item) ? 'skill-read-entry' : 'command-entry';
  if (item.type === 'fileChange') return 'file-entry';
  return 'tool-entry';
}

function isSearchPageItem(item: ThreadItem) {
  return item.type === 'webSearch' && webSearchActionType(item) !== 'openPage';
}

function createWebSearchGroup(items: ThreadItem[]): DisplayThreadItem | null {
  const queries = items
    .filter(isSearchPageItem)
    .map(item => webSearchText(item))
    .filter(Boolean);
  if (!queries.length) return null;
  const source = items.find(isSearchPageItem)!;
  return {
    ...source,
    id: `web-search-group-${source.id}-${queries.length}`,
    type: 'webSearchGroup',
    memberIds: items.filter(isSearchPageItem).map(item => item.id),
    searchQueries: queries
  } as DisplayThreadItem;
}

function compactProcessStage(items: ThreadItem[]): DisplayThreadItem[] {
  if (!items.length) return [];
  const searchGroup = createWebSearchGroup(items);
  if (searchGroup) return [searchGroup];

  const last = items[items.length - 1];
  if (!isSkillCommandItem(last)) return [last];

  let start = items.length - 1;
  while (start > 0 && isSkillCommandItem(items[start - 1])) start -= 1;
  const skillGroup = createSkillReadGroup(items.slice(start));
  return skillGroup ? [skillGroup] : [];
}

function compactProcessItemsByStage(items: ThreadItem[]): DisplayThreadItem[] {
  const result: ThreadItem[] = [];
  let stageItems: ThreadItem[] = [];
  const flushStage = () => {
    result.push(...compactProcessStage(stageItems));
    stageItems = [];
  };

  for (const item of items) {
    if (item.type === 'agentMessage' && isIntermediateAgentMessage(item)) {
      flushStage();
      result.push(item);
      continue;
    }
    stageItems.push(item);
  }
  flushStage();
  return result;
}
