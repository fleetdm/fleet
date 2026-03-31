import createMockUser from "__mocks__/userMock";

import { sortAvailableTeams } from "./app";

describe("sortAvailableTeams", () => {
  it("places Unassigned last for global team users", () => {
    const teams = [
      { id: 0, name: "Unassigned" },
      { id: 2, name: "Zebra" },
      { id: 1, name: "Alpha" },
      { id: -1, name: "All fleets" },
    ];
    const result = sortAvailableTeams(teams, createMockUser());
    expect(result.map((t) => t.name)).toEqual([
      "All fleets",
      "Alpha",
      "Zebra",
      "Unassigned",
    ]);
  });

  it("does not include All fleets or Unassigned for non-global users", () => {
    const teams = [
      { id: 0, name: "Unassigned" },
      { id: 2, name: "Zebra" },
      { id: 1, name: "Alpha" },
      { id: -1, name: "All fleets" },
    ];
    const result = sortAvailableTeams(
      teams,
      createMockUser({ global_role: null })
    );
    expect(result.map((t) => t.name)).toEqual(["Alpha", "Zebra"]);
  });

  it("sorts named teams alphabetically (case-insensitive)", () => {
    const teams = [
      { id: 3, name: "charlie" },
      { id: 1, name: "Alpha" },
      { id: 2, name: "Bravo" },
    ];
    const result = sortAvailableTeams(
      teams,
      createMockUser({ global_role: null })
    );
    expect(result.map((t) => t.name)).toEqual(["Alpha", "Bravo", "charlie"]);
  });
});
