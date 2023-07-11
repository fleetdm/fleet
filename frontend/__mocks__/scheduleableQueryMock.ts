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
  interval: 3600,
  platform: "macos,windows,linux",
  min_osquery_version: "",
  automations_enabled: true,
  logging: "snapshot",
  saved: true,
  author_id: 1,
  author_name: "Test User",
  author_email: "test@example.com",
  observer_can_run: false,
  packs: [],
};

const createMockSchedulableQuery = (
  overrides?: Partial<ISchedulableQuery>
): ISchedulableQuery => {
  return { ...DEFAULT_SCHEDULABLE_QUERY_MOCK, ...overrides };
};

export default createMockSchedulableQuery;
