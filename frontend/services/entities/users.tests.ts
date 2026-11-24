import { toFleetRoles } from "./users";

describe("toFleetRoles", () => {
  it("keeps only the id and role of each fleet", () => {
    const fleets = [
      {
        id: 3,
        role: "admin",
        name: "Fleet A",
        mdm: {
          windows_settings: { custom_settings: [], configuration_profiles: [] },
        },
      },
    ];

    expect(toFleetRoles(fleets)).toEqual([{ id: 3, role: "admin" }]);
  });

  it("returns undefined when there are no fleets", () => {
    expect(toFleetRoles(undefined)).toBeUndefined();
  });
});
