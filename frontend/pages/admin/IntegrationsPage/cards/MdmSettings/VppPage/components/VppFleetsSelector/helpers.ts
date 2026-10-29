import { IMdmVppToken } from "interfaces/mdm";

export interface IVppFleetsValue {
  allFleets: boolean;
  fleetIds: number[];
}

export const valueFromToken = (token: IMdmVppToken): IVppFleetsValue => ({
  allFleets: token.teams?.length === 0,
  fleetIds: token.teams?.map((t) => t.team_id) ?? [],
});

/** The PATCH body's fleets: [] means All fleets, null means unassigned. */
export const fleetsForApi = ({
  allFleets,
  fleetIds,
}: IVppFleetsValue): number[] | null => {
  if (allFleets) return [];
  return fleetIds.length ? fleetIds : null;
};
