export interface Conversation {
  id: string;
  title: string;
  status: 'active' | 'archived';
  pinnedAt?: string;
  archivedAt?: string;
  groupId?: string;
  running: boolean;
  runningStartedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ConversationMetadata {
  id: string;
  title: string;
  updatedAt: string;
}

export interface ThreadItem {
  id: string;
  type: string;
  text?: string;
  phase?: string | null;
  content?: Array<{ type?: string; text?: string; url?: string; path?: string }> | string[];
  summary?: string[];
  command?: string;
  aggregatedOutput?: string | null;
  cwd?: string;
  exitCode?: number | null;
  status?: string;
  tool?: string;
  server?: string;
  pluginId?: string | null;
  commandActions?: Array<
    | { type: 'read'; command: string; name: string; path: string }
    | { type: 'listFiles'; command: string; path: string | null }
    | { type: 'search'; command: string; query: string | null; path: string | null }
    | { type: 'unknown'; command: string }
  >;
  arguments?: unknown;
  result?: unknown;
  query?: string;
  changes?: Array<Record<string, unknown>>;
  durationMs?: number | null;
  dataUrl?: string;
  alt?: string;
  savedPath?: string | null;
  [key: string]: unknown;
}

export interface Turn {
  id: string;
  status: string;
  startedAt?: string;
  completedAt?: string;
  durationMs?: number;
  error?: string;
  items: ThreadItem[];
}

export interface GeneratedImage {
  id: string;
  dataUrl: string;
  alt: string;
}

export interface ConversationDetail {
  conversation: Conversation;
  turns: Turn[];
}

export interface StreamEvent {
  type: string;
  conversationId: string;
  turnId?: string;
  data?: unknown;
  createdAt: string;
}

export interface Envelope<T> {
  code: number;
  data: T;
  msg: string;
}

export interface EventsResponse {
  event: string;
  data: string;
}

export interface StartedTurn {
  id: string;
  startedAt: string;
}

export interface WorkspaceEntry {
  name: string;
  path: string;
  type: 'directory' | 'file';
  size: number;
}

export interface WorkspaceFileContent {
  path: string;
  name: string;
  content: string;
  contentType: string;
}

export interface WorkspaceFileBlob {
  path: string;
  name: string;
  blob: Blob;
  contentType: string;
}

export interface ConversationGroup {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}


export interface AgentApiKey {
  id: string;
  keyHint: string;
  createdAt: string;
}

export interface CreatedAgentApiKey extends AgentApiKey {
  key: string;
}

export interface LoginResponse {
  token: string;
  refreshToken: string;
}

export interface UserInfo {
  userUuid: string;
  username: string;
}
