import { IHostAISession, IHostAISessionEvent } from "interfaces/ai_sessions";
import {
  IGetHostAISessionEventsResponse,
  IGetHostAISessionsResponse,
} from "services/entities/hosts";

const DEFAULT_HOST_AI_SESSION_MOCK: IHostAISession = {
  id: 1,
  host_id: 1,
  agent: "claude_code",
  agent_version: "2.1.3",
  agent_session_id: "0b6f3a51-4f3e-4f39-9d4c-2f1d2a9a6c11",
  username: "alice",
  account_identifier: "alice@example.com",
  model: "claude-opus-4-1",
  project_path: "/Users/alice/src/app",
  git_repository: "github.com/example/app",
  git_branch: "main",
  started_at: "2026-10-20T10:00:00Z",
  ended_at: "2026-10-20T11:05:30Z",
  user_message_count: 4,
  assistant_message_count: 12,
  tool_call_count: 7,
  categories: ["command", "file_access"],
  backfill: false,
  input_token_count: 12345,
  output_token_count: 6789,
  cache_read_token_count: 0,
  cache_write_token_count: 0,
  transcript_path: "/Users/alice/.claude/projects/app/session.jsonl",
  transcript_size: 20480,
};

export const createMockHostAISession = (
  overrides?: Partial<IHostAISession>
): IHostAISession => ({ ...DEFAULT_HOST_AI_SESSION_MOCK, ...overrides });

export const createMockGetHostAISessionsResponse = (
  overrides?: Partial<IGetHostAISessionsResponse>
): IGetHostAISessionsResponse => ({
  ai_sessions: [createMockHostAISession()],
  meta: { has_next_results: false, has_previous_results: false },
  count: 1,
  ...overrides,
});

const DEFAULT_HOST_AI_SESSION_EVENT_MOCK: IHostAISessionEvent = {
  id: 1,
  categories: ["command"],
  target: "npm test",
  decision: "allowed",
  executed_at: "2026-10-20T10:01:00Z",
  payload: { name: "Bash", id: "toolu_01", input: { command: "npm test" } },
  truncated: false,
};

export const createMockHostAISessionEvent = (
  overrides?: Partial<IHostAISessionEvent>
): IHostAISessionEvent => ({
  ...DEFAULT_HOST_AI_SESSION_EVENT_MOCK,
  ...overrides,
});

export const createMockGetHostAISessionEventsResponse = (
  overrides?: Partial<IGetHostAISessionEventsResponse>
): IGetHostAISessionEventsResponse => ({
  events: [createMockHostAISessionEvent()],
  meta: { has_next_results: false, has_previous_results: false },
  count: 1,
  ...overrides,
});
