import createMockUser from "__mocks__/userMock";

import permissions from ".";

describe("permissions - isAdminForAllUserTeams", () => {
  const globalAdmin = createMockUser({ global_role: "admin", teams: [] });
  const teamAdminTeam1 = createMockUser({
    global_role: null,
    teams: [{ id: 1, name: "Team 1", role: "admin" }],
  });
  const teamAdminTeam1And2 = createMockUser({
    global_role: null,
    teams: [
      { id: 1, name: "Team 1", role: "admin" },
      { id: 2, name: "Team 2", role: "admin" },
    ],
  });
  const teamAdminTeam1ObserverTeam2 = createMockUser({
    global_role: null,
    teams: [
      { id: 1, name: "Team 1", role: "admin" },
      { id: 2, name: "Team 2", role: "observer" },
    ],
  });

  it("returns false when there is no current user", () => {
    const target = createMockUser({
      global_role: null,
      teams: [{ id: 1, name: "Team 1", role: "observer" }],
    });
    expect(permissions.isAdminForAllUserTeams(null, target)).toBe(false);
  });

  it("allows a global admin to manage any user", () => {
    const target = createMockUser({
      global_role: null,
      teams: [
        { id: 1, name: "Team 1", role: "observer" },
        { id: 2, name: "Team 2", role: "admin" },
      ],
    });
    expect(permissions.isAdminForAllUserTeams(globalAdmin, target)).toBe(true);
  });

  it("allows a team admin to manage a user that only belongs to fleets they administer", () => {
    const target = createMockUser({
      global_role: null,
      teams: [{ id: 1, name: "Team 1", role: "observer" }],
    });
    expect(permissions.isAdminForAllUserTeams(teamAdminTeam1, target)).toBe(
      true
    );
  });

  it("prevents a team admin from managing a user that also belongs to fleets they do not administer", () => {
    const target = createMockUser({
      global_role: null,
      teams: [
        { id: 1, name: "Team 1", role: "observer" },
        { id: 2, name: "Team 2", role: "admin" },
      ],
    });
    expect(permissions.isAdminForAllUserTeams(teamAdminTeam1, target)).toBe(
      false
    );
  });

  it("allows a team admin of all the user's fleets to manage that user", () => {
    const target = createMockUser({
      global_role: null,
      teams: [
        { id: 1, name: "Team 1", role: "observer" },
        { id: 2, name: "Team 2", role: "admin" },
      ],
    });
    expect(permissions.isAdminForAllUserTeams(teamAdminTeam1And2, target)).toBe(
      true
    );
  });

  it("prevents a team admin of one fleet but only observer of another from managing a user that belongs to both", () => {
    const target = createMockUser({
      global_role: null,
      teams: [
        { id: 1, name: "Team 1", role: "observer" },
        { id: 2, name: "Team 2", role: "observer" },
      ],
    });
    expect(
      permissions.isAdminForAllUserTeams(teamAdminTeam1ObserverTeam2, target)
    ).toBe(false);
  });

  it("prevents a team admin from managing a user with a global role", () => {
    const target = createMockUser({
      global_role: "observer",
      teams: [{ id: 1, name: "Team 1", role: "observer" }],
    });
    expect(permissions.isAdminForAllUserTeams(teamAdminTeam1, target)).toBe(
      false
    );
  });

  it("prevents a team admin from managing a user with no fleets", () => {
    const target = createMockUser({ global_role: null, teams: [] });
    expect(permissions.isAdminForAllUserTeams(teamAdminTeam1, target)).toBe(
      false
    );
  });
});

describe("permissions - canWriteSoftware", () => {
  // Mirrors backend WRITE on `SoftwareInstaller` (policy.rego L827-832, L842-848):
  // admin | maintainer | gitops are allowed. The UI doesn't surface gitops users,
  // so the helper returns true only for admin / maintainer (global or team-scoped).
  const TEAM_ID = 1;

  it("returns false when there is no user", () => {
    expect(permissions.canWriteSoftware(null, TEAM_ID)).toBe(false);
  });

  it("allows a global admin regardless of team", () => {
    const user = createMockUser({ global_role: "admin", teams: [] });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(true);
    expect(permissions.canWriteSoftware(user, null)).toBe(true);
  });

  it("allows a global maintainer regardless of team", () => {
    const user = createMockUser({ global_role: "maintainer", teams: [] });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(true);
  });

  it("allows a team admin on their team", () => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: TEAM_ID, name: "Team 1", role: "admin" }],
    });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(true);
  });

  it("allows a team maintainer on their team", () => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: TEAM_ID, name: "Team 1", role: "maintainer" }],
    });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(true);
  });

  it("denies a team admin on a different team", () => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: 2, name: "Team 2", role: "admin" }],
    });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(false);
  });

  it.each([
    ["technician", "technician"],
    ["observer", "observer"],
    ["observer_plus", "observer_plus"],
  ] as const)("denies a global %s", (_label, role) => {
    const user = createMockUser({ global_role: role, teams: [] });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(false);
  });

  it.each([
    ["technician", "technician"],
    ["observer", "observer"],
    ["observer_plus", "observer_plus"],
  ] as const)("denies a team %s on their team", (_label, role) => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: TEAM_ID, name: "Team 1", role }],
    });
    expect(permissions.canWriteSoftware(user, TEAM_ID)).toBe(false);
  });
});

describe("permissions - canDownloadSoftwareInstaller", () => {
  // Mirrors backend READ on `installable_entity` (policy.rego L837-865):
  // admin | maintainer | technician | gitops are allowed at both global and
  // team scope. Observer / observer+ are excluded. Guards the download button
  // in the UI so observers don't see it and hit the backend 403.
  const TEAM_ID = 1;

  it("returns false when there is no user", () => {
    expect(permissions.canDownloadSoftwareInstaller(null, TEAM_ID)).toBe(false);
  });

  it.each([
    ["admin", "admin"],
    ["maintainer", "maintainer"],
    ["technician", "technician"],
  ] as const)("allows a global %s", (_label, role) => {
    const user = createMockUser({ global_role: role, teams: [] });
    expect(permissions.canDownloadSoftwareInstaller(user, TEAM_ID)).toBe(true);
    expect(permissions.canDownloadSoftwareInstaller(user, null)).toBe(true);
  });

  it.each([
    ["admin", "admin"],
    ["maintainer", "maintainer"],
    ["technician", "technician"],
  ] as const)("allows a team %s on their team", (_label, role) => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: TEAM_ID, name: "Team 1", role }],
    });
    expect(permissions.canDownloadSoftwareInstaller(user, TEAM_ID)).toBe(true);
  });

  it("denies a team technician on a different team", () => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: 2, name: "Team 2", role: "technician" }],
    });
    expect(permissions.canDownloadSoftwareInstaller(user, TEAM_ID)).toBe(false);
  });

  it.each([
    ["observer", "observer"],
    ["observer_plus", "observer_plus"],
  ] as const)("denies a global %s", (_label, role) => {
    const user = createMockUser({ global_role: role, teams: [] });
    expect(permissions.canDownloadSoftwareInstaller(user, TEAM_ID)).toBe(false);
  });

  it.each([
    ["observer", "observer"],
    ["observer_plus", "observer_plus"],
  ] as const)("denies a team %s on their team", (_label, role) => {
    const user = createMockUser({
      global_role: null,
      teams: [{ id: TEAM_ID, name: "Team 1", role }],
    });
    expect(permissions.canDownloadSoftwareInstaller(user, TEAM_ID)).toBe(false);
  });
});
