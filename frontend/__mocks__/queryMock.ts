import { ISchedulableQuery } from "interfaces/schedulable_query";

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
  interval: 300,
  packs: [],
  team_id: null,
  platform: "",
  min_osquery_version: "",
  automations_enabled: false,
  logging: "snapshot",
  stats: undefined,
};

const createMockQuery = (
  overrides?: Partial<ISchedulableQuery>
): ISchedulableQuery => {
  return { ...DEFAULT_QUERY_MOCK, ...overrides };
};

export default createMockQuery;
