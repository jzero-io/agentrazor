import type { ThreadItem } from '../service/api';

export interface CodexTurnPayload {
  id?: unknown;
  status?: unknown;
  startedAt?: unknown;
  completedAt?: unknown;
  durationMs?: unknown;
  items?: unknown;
  error?: unknown;
}

export interface TextDeltaParams {
  delta?: string;
  itemId?: string;
  contentIndex?: number;
}

export interface AgentMessageDeltaParams {
  delta?: string;
  itemId?: string;
  phase?: string | null;
  item?: ThreadItem;
}

export function eventParams<T extends object>(data: unknown) {
  return (data as { params?: T } | undefined)?.params;
}

export function codexTurnPayload(data: unknown) {
  return eventParams<{ turn?: CodexTurnPayload }>(data)?.turn;
}

export function codexTurnError(turn: CodexTurnPayload | undefined) {
  if (!turn?.error || typeof turn.error !== 'object') return '';
  const message = (turn.error as { message?: unknown }).message;
  return typeof message === 'string' ? message : '';
}

export function taskPayload(data: unknown) {
  const params = eventParams<Record<string, unknown>>(data);
  return ((params?.msg || params?.payload || params) ?? {}) as Record<string, unknown>;
}

export function textDeltaParams(data: unknown) {
  return eventParams<TextDeltaParams>(data);
}

export function agentMessageDeltaParams(data: unknown) {
  return eventParams<AgentMessageDeltaParams>(data);
}

export function itemParams(data: unknown) {
  return eventParams<{ item?: ThreadItem }>(data);
}

export function unixSecondsToIso(value: unknown) {
  const seconds = Number(value);
  return Number.isFinite(seconds) && seconds > 0 ? new Date(seconds * 1000).toISOString() : undefined;
}

export function nonNegativeNumber(value: unknown) {
  const number = Number(value);
  return Number.isFinite(number) && number >= 0 ? number : undefined;
}
