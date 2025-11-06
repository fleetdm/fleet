import { IUser } from "interfaces/user";

const DEFAULT_USER_MOCK: IUser = {
  created_at: "2022-01-01T12:00:00Z",
  updated_at: "2022-01-02T12:00:00Z",
  id: 1,
  name: "Test User",
  email: "testUser@test.com",
  role: "admin",
  force_password_reset: false,
  gravatar_url: "http://test.com",
  sso_enabled: false,
  global_role: "admin",
  api_only: false,
  teams: [],
};

const createMockUser = (overrides?: Partial<IUser>): IUser => {
  return { ...DEFAULT_USER_MOCK, ...overrides };
};

export default createMockUser;
