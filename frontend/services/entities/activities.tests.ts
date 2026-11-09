import { withFleetDetailKeys } from "./activities";

describe("withFleetDetailKeys", () => {
  it("renames the old detail keys of activities recorded before the rename", () => {
    const activity = {
      id: 1,
      details: {
        team_id: 3,
        team_name: "Workstations",
        teams: [{ id: 3, name: "Workstations" }],
        role: "admin",
      },
    };

    expect(withFleetDetailKeys(activity)).toEqual({
      id: 1,
      details: {
        fleet_id: 3,
        fleet_name: "Workstations",
        fleets: [{ id: 3, name: "Workstations" }],
        role: "admin",
      },
    });
  });

  it("keeps the fleet keys and drops the old ones when both are present", () => {
    const activity = {
      details: { team_id: 1, team_name: "Old", fleet_id: 2, fleet_name: "New" },
    };

    expect(withFleetDetailKeys(activity).details).toEqual({
      fleet_id: 2,
      fleet_name: "New",
    });
  });

  it("returns activities without details unchanged", () => {
    const activity = { id: 7, details: null };

    expect(withFleetDetailKeys(activity)).toBe(activity);
  });
});
