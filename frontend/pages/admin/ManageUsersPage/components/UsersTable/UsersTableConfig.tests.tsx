import createMockUser from "__mocks__/userMock";
import { IInvite } from "interfaces/invite";

import { combineDataSets } from "./UsersTableConfig";

const daysAgo = (days: number): string =>
  new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString();

const createMockInvite = (overrides?: Partial<IInvite>): IInvite => ({
  created_at: daysAgo(1),
  updated_at: daysAgo(1),
  id: 1,
  invited_by: 99,
  email: "invitee@example.com",
  name: "Invited User",
  sso_enabled: false,
  global_role: "observer",
  teams: [],
  ...overrides,
});

describe("UsersTableConfig - combineDataSets", () => {
  it("returns 'Active' for a user who logged in recently", () => {
    const users = [createMockUser({ last_login_at: daysAgo(1) })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Active");
  });

  it("returns 'Inactive' for a user who hasn't logged in for 30+ days", () => {
    const users = [createMockUser({ last_login_at: daysAgo(31) })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Inactive");
  });

  it("falls back to created_at for users who have never logged in", () => {
    const stale = createMockUser({
      last_login_at: null,
      created_at: daysAgo(31),
    });
    const fresh = createMockUser({
      id: 2,
      last_login_at: null,
      created_at: daysAgo(1),
    });
    const [staleRow, freshRow] = combineDataSets([stale, fresh], [], 99);
    expect(staleRow.status).toBe("Inactive");
    expect(freshRow.status).toBe("Active");
  });

  it("never returns 'Inactive' for API-only users", () => {
    const users = [
      createMockUser({
        api_only: true,
        last_login_at: daysAgo(100),
        created_at: daysAgo(100),
      }),
    ];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Active");
  });

  it("returns 'No access' for a user without a role, regardless of last login", () => {
    const users = [
      createMockUser({
        global_role: null,
        teams: [],
        last_login_at: daysAgo(31),
      }),
    ];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("No access");
  });

  it("returns 'Invite pending' for invites", () => {
    const invites = [createMockInvite()];
    const [row] = combineDataSets([], invites, 99);
    expect(row.status).toBe("Invite pending");
  });
});
