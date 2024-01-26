// "SchedulableQuery" to be used in developing frontend for #7765

import { ISchedulableQuery } from "interfaces/schedulable_query";

const DEFAULT_SCHEDULABLE_QUERY_MOCK: ISchedulableQuery = {
  created_at: "2022-11-03T17:22:14Z",
  updated_at: "2022-11-03T17:22:14Z",
  id: 1,
  name: "Test Query",
  description: "A test query",
  query: "SELECT * FROM users",
  team_id: null,
  interval: 43200, // Every 12 hours
  platform: "darwin,windows,linux",
  min_osquery_version: "",
  automations_enabled: true,
  logging: "snapshot",
  saved: true,
  author_id: 1,
  author_name: "Test User",
  author_email: "test@example.com",
  observer_can_run: false,
  discard_data: false,
  packs: [],
  stats: {
    system_time_p50: 28.1053,
    system_time_p95: 397.6667,
    user_time_p50: 29.9412,
    user_time_p95: 251.4615,
    total_executions: 5746,
  },
  editingExistingQuery: false,
};

const createMockSchedulableQuery = (
  overrides?: Partial<ISchedulableQuery>
): ISchedulableQuery => {
  return { ...DEFAULT_SCHEDULABLE_QUERY_MOCK, ...overrides };
};

export default createMockSchedulableQuery;
