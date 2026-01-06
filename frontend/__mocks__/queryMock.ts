import { ISchedulableQuery } from "interfaces/schedulable_query";
import { IQueryStats } from "interfaces/query_stats";

const DEFAULT_QUERY_MOCK: ISchedulableQuery = {
  created_at: "2022-11-03T17:22:14Z",
  updated_at: "2022-11-03T17:22:14Z",
  id: 1,
  name: "Test Query",
  description: "A test query",
  query: "SELECT * FROM users",
  saved: true,
  author_id: 1,
  author_name: "Test User",
  author_email: "test@example.com",
  observer_can_run: false,
  discard_data: false,
  interval: 300,
  packs: [],
  team_id: null,
  platform: "",
  min_osquery_version: "",
  automations_enabled: false,
  logging: "snapshot",
  stats: {
    user_time_p50: 0,
    user_time_p95: 2,
    system_time_p50: 0,
    system_time_p95: 1,
    total_executions: 6,
  },
  editingExistingQuery: false,
};

const createMockQuery = (
  overrides?: Partial<ISchedulableQuery>
): ISchedulableQuery => {
  return { ...DEFAULT_QUERY_MOCK, ...overrides };
};

const DEFAULT_QUERY_STATS_MOCK: IQueryStats = {
  scheduled_query_name: "test-query",
  scheduled_query_id: 1,
  query_name: "Test Query",
  discard_data: false,
  last_fetched: "2025-01-01T00:00:00Z",
  automations_enabled: false,
  description: "A test query",
  pack_name: "test-pack",
  pack_id: 1,
  average_memory: 100,
  denylisted: false,
  executions: 10,
  interval: 3600,
  last_executed: "2025-01-01T00:00:00Z",
  output_size: 1024,
  system_time: 50,
  user_time: 100,
  wall_time: 150,
};

export const createMockQueryStats = (
  overrides?: Partial<IQueryStats>
): IQueryStats => {
  return { ...DEFAULT_QUERY_STATS_MOCK, ...overrides };
};

export default createMockQuery;
