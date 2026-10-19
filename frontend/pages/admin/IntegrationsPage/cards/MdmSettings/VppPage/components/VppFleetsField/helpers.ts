import { IMdmVppToken } from "interfaces/mdm";

export type VppFleetsMode = "all" | "custom";

export interface IVppFleetsValue {
  mode: VppFleetsMode;
  teamIds: number[];
}

export const DEFAULT_VPP_FLEETS_VALUE: IVppFleetsValue = {
  mode: "all",
  teamIds: [],
};

/** A token with `teams: []` is on all fleets, and `teams: null` is unassigned. */
export const vppFleetsValueFromToken = (
  token: IMdmVppToken
): IVppFleetsValue => {
  if (token.teams?.length === 0) {
    return { mode: "all", teamIds: [] };
  }
  return {
    mode: "custom",
    teamIds: token.teams?.map((team) => team.team_id) ?? [],
  };
};

/**
 * Returns the `fleets` value for the API: `[]` assigns the token to all fleets,
 * and `null` leaves it unassigned.
 */
export const teamIdsForApi = (value: IVppFleetsValue): number[] | null => {
  if (value.mode === "all") {
    return [];
  }
  return value.teamIds.length ? value.teamIds : null;
};
