import { IQuery } from "interfaces/query";

const DEFAULT_QUERY_MOCK: IQuery = {
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
  packs: [],
};

const createMockQuery = (overrides?: Partial<IQuery>): IQuery => {
  return { ...DEFAULT_QUERY_MOCK, ...overrides };
};

export default createMockQuery;
