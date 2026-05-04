import { preferredOrLowestIdFleet } from "./useTeamIdParam";

describe("preferredOrLowestIdFleet", () => {
  it('returns "Workstations" when present', () => {
    const fleets = [
      { id: 1, name: "Alpha" },
      { id: 2, name: "Workstations" },
      { id: 3, name: "Zebra" },
    ];
    expect(preferredOrLowestIdFleet(fleets)).toEqual({
      id: 2,
      name: "Workstations",
    });
  });

  it('returns "💻 Workstations" (emoji variant) when present', () => {
    const fleets = [
      { id: 1, name: "Alpha" },
      { id: 3, name: "💻 Workstations" },
    ];
    expect(preferredOrLowestIdFleet(fleets)).toEqual({
      id: 3,
      name: "💻 Workstations",
    });
  });

  it("prefers plain over emoji when both exist (first match wins)", () => {
    const fleets = [
      { id: 5, name: "Workstations" },
      { id: 2, name: "💻 Workstations" },
    ];
    expect(preferredOrLowestIdFleet(fleets)).toEqual({
      id: 5,
      name: "Workstations",
    });
  });

  it("matches case-insensitively", () => {
    const fleets = [
      { id: 1, name: "Alpha" },
      { id: 4, name: "WORKSTATIONS" },
    ];
    expect(preferredOrLowestIdFleet(fleets)).toEqual({
      id: 4,
      name: "WORKSTATIONS",
    });
  });

  it("falls back to lowest ID when no Workstations fleet exists", () => {
    const fleets = [
      { id: 10, name: "Zebra" },
      { id: 3, name: "Charlie" },
      { id: 7, name: "Mike" },
    ];
    expect(preferredOrLowestIdFleet(fleets)).toEqual({
      id: 3,
      name: "Charlie",
    });
  });

  it("returns undefined for an empty array", () => {
    expect(preferredOrLowestIdFleet([])).toBeUndefined();
  });
});
