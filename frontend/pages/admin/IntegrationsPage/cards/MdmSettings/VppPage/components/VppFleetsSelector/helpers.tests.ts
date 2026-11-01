import { createMockVppToken } from "__mocks__/mdmMock";
import { IMdmVppToken } from "interfaces/mdm";

import { fleetsForApi, valueFromToken } from "./helpers";

const token = (teams: IMdmVppToken["teams"]): IMdmVppToken =>
  createMockVppToken({ teams });

describe("VppFleetsSelector helpers", () => {
  describe("valueFromToken", () => {
    it("maps an empty teams array to All fleets", () => {
      expect(valueFromToken(token([]))).toEqual({
        allFleets: true,
        fleetIds: [],
      });
    });

    it("maps null teams to Custom with no fleets", () => {
      expect(valueFromToken(token(null))).toEqual({
        allFleets: false,
        fleetIds: [],
      });
    });

    it("maps teams to Custom with fleet ids, including Unassigned", () => {
      expect(
        valueFromToken(
          token([
            { team_id: 0, name: "No team" },
            { team_id: 2, name: "Pirates" },
          ])
        )
      ).toEqual({ allFleets: false, fleetIds: [0, 2] });
    });
  });

  describe("fleetsForApi", () => {
    it("returns [] for All fleets, even with remembered fleet ids", () => {
      expect(fleetsForApi({ allFleets: true, fleetIds: [1, 2] })).toEqual([]);
    });

    it("returns null for Custom with no fleets", () => {
      expect(fleetsForApi({ allFleets: false, fleetIds: [] })).toBeNull();
    });

    it("returns the ids for Custom with fleets", () => {
      expect(fleetsForApi({ allFleets: false, fleetIds: [0, 3] })).toEqual([
        0,
        3,
      ]);
    });
  });
});
