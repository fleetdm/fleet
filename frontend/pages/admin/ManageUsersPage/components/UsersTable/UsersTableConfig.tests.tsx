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
  it("maps the server-computed 'active' status to 'Active'", () => {
    const users = [createMockUser({ status: "active" })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Active");
  });

  it("maps the server-computed 'inactive' status to 'Inactive'", () => {
    const users = [createMockUser({ status: "inactive" })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Inactive");
  });

  it("maps the server-computed 'no_access' status to 'No access'", () => {
    const users = [createMockUser({ status: "no_access" })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("No access");
  });

  it("defaults to 'Active' when the server did not compute a status", () => {
    const users = [createMockUser({ status: undefined })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Active");
  });

  it("returns 'Invite pending' for invites", () => {
    const invites = [createMockInvite()];
    const [row] = combineDataSets([], invites, 99);
    expect(row.status).toBe("Invite pending");
  });

  it("returns 'No access' for invites without a role", () => {
    const invites = [createMockInvite({ global_role: null, teams: [] })];
    const [row] = combineDataSets([], invites, 99);
    expect(row.status).toBe("No access");
  });
});
