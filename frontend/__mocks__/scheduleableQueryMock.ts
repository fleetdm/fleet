// "ScheduleableQuery" to be used in developing frontend for #7765

import { IScheduleableQuery } from "interfaces/scheduleable_query";

const DEFAULT_SCHEDULEABLE_QUERY_MOCK: IScheduleableQuery = {
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

const createMockScheduleableQuery = (
  overrides?: Partial<IScheduleableQuery>
): IScheduleableQuery => {
  return { ...DEFAULT_SCHEDULEABLE_QUERY_MOCK, ...overrides };
};

export default createMockScheduleableQuery;
