import { useContext } from "react";

import { AppContext } from "context/app";
import { ITeam } from "interfaces/team";
import { Role } from "interfaces/role";

import {
  FreePermissions,
  PremiumPermissions,
  freePermissions,
  premiumPermissions,
} from "./permissions";

/** role names as they are used to check permissions in the UI */
export type PermissionRole =
  | "global-admin"
  | "global-maintainer"
  | "global-observer"
  | "global-observer-plus"
  | "global-gitops"
  | "team-admin"
  | "team-maintainer"
  | "team-observer"
  | "team-observer-plus"
  | "team-gitops";

// These are mappings of the API role names to the role names used to check
const ApiToClientGlobalRoleMap: Record<Role, PermissionRole> = {
  admin: "global-admin",
  maintainer: "global-maintainer",
  observer: "global-observer",
  observer_plus: "global-observer-plus",
  gitops: "global-gitops",
};
const ApiToClientTeamRoleMap: Record<Role, PermissionRole> = {
  admin: "team-admin",
  maintainer: "team-maintainer",
  observer: "team-observer",
  observer_plus: "team-observer-plus",
  gitops: "team-gitops",
};

const getCurrentTeamRole = (
  currentUserTeams: ITeam[],
  currentTeamId: number
) => {
  if (!currentUserTeams || !currentTeamId) {
    return undefined;
  }

  const currentTeam = currentUserTeams.find(
    (team) => team.id === currentTeamId
  );

  return currentTeam?.role;
};

const getTierPermissionMap = (isPremiumTier: boolean): any => {
  return isPremiumTier ? premiumPermissions : freePermissions;
};

export const usePermissions = () => {
  const { currentUser, currentTeam, isPremiumTier } = useContext(AppContext);

  // Quick exit if we don't have the data we need to check permissions.
  if (!currentUser || !currentTeam || !isPremiumTier)
    return { hasPermission: () => false };

  const globalRole = currentUser.global_role;
  const currentTeamRole = getCurrentTeamRole(currentUser.teams, currentTeam.id);

  const userTierPermissionMap = getTierPermissionMap(isPremiumTier);

  const hasGlobalPermission = (
    permissionName: FreePermissions | PremiumPermissions
  ) => {
    return (
      globalRole !== null &&
      globalRole !== undefined &&
      userTierPermissionMap[permissionName].includes(
        ApiToClientGlobalRoleMap[globalRole]
      )
    );
  };

  const hasTeamPermission = (
    permissionName: FreePermissions | PremiumPermissions
  ) => {
    return (
      currentTeamRole !== null &&
      currentTeamRole !== undefined &&
      userTierPermissionMap[permissionName].includes(
        ApiToClientTeamRoleMap[currentTeamRole]
      )
    );
  };

  const hasPermission = (
    permissionName: FreePermissions | PremiumPermissions
  ) => {
    return (
      hasGlobalPermission(permissionName) || hasTeamPermission(permissionName)
    );
  };

  return { hasPermission };
};
