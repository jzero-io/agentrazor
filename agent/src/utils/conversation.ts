import type { Conversation } from '../service/api';

export function conversationCreatedAtMs(item: Conversation) {
  const time = Date.parse(item.createdAt || '');
  return Number.isFinite(time) ? time : 0;
}

export function sortConversationsByCreatedAt(items: Conversation[]) {
  return [...items].sort((left, right) => conversationCreatedAtMs(right) - conversationCreatedAtMs(left));
}

export function displayConversationTitle(item?: Pick<Conversation, 'title'> | null) {
  return item?.title?.trim() || '新对话';
}

export function mergeConversationSnapshot(current: Conversation | undefined, incoming: Conversation): Conversation {
  const title = incoming.title.trim() || current?.title?.trim() || '';
  return { ...current, ...incoming, title };
}

export function conversationIdFromPath(pathname: string): string {
  const match = pathname.match(/^\/c\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : '';
}

export function messagePreview(content: string) {
  const summary = content.replace(/\s+/g, ' ').trim();
  return summary.length > 72 ? `${summary.slice(0, 72)}…` : summary;
}

export function formatConversationDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? ''
    : new Intl.DateTimeFormat('zh-CN', {
        year: 'numeric', month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit'
      }).format(date);
}

function isSameDay(left: Date, right: Date) {
  return left.getFullYear() === right.getFullYear()
    && left.getMonth() === right.getMonth()
    && left.getDate() === right.getDate();
}

export function formatMessageTime(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const now = new Date();
  const options: Intl.DateTimeFormatOptions = isSameDay(date, now)
    ? { hour: '2-digit', minute: '2-digit' }
    : date.getFullYear() === now.getFullYear()
      ? { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }
      : { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' };
  return new Intl.DateTimeFormat('zh-CN', options).format(date);
}
