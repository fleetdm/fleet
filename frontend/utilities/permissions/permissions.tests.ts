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
