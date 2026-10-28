import { IListSort } from "./list_options";

export type AISessionCategory = "command" | "file_access";

export interface IHostAISession {
  id: number;
  host_id: number;
  agent: string;
  agent_version: string;
  agent_session_id: string;
  username: string;
  account_identifier: string;
  model: string;
  project_path: string;
  git_repository: string;
  git_branch: string;
  started_at: string | null;
  ended_at: string | null;
  user_message_count: number;
  assistant_message_count: number;
  tool_call_count: number;
  categories: AISessionCategory[];
  /** Fleet is still collecting the calls the session made before Fleet first
   * stored it, so fewer events than tool_call_count may be listed. */
  backfill: boolean;
  input_token_count: number;
  output_token_count: number;
  cache_read_token_count: number;
  cache_write_token_count: number;
  transcript_path: string;
  transcript_size: number;
}

export interface IHostAISessionEvent {
  id: number;
  categories: AISessionCategory[];
  target: string;
  decision: string;
  executed_at: string | null;
  payload: unknown;
  /** fleetd cut the call's arguments before they reached the payload. */
  truncated: boolean;
}

export const AI_SESSIONS_DEFAULT_SORT: IListSort = {
  order_key: "started_at",
  order_direction: "desc",
};
