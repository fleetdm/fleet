import { AppContext } from "context/app";
import { ITeam } from "interfaces/team";
import { useContext } from "react";

/** global roles as they come back from the API. */
export type GlobalRole =
  | "admin"
  | "maintainer"
  | "observer"
  | "observer_plus"
  | "gitops";
/** team roles as they come back from the API. */
export type TeamRole =
  | "admin"
  | "maintainer"
  | "observer"
  | "observer_plus"
  | "gitops";

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
const ApiToClientGlobalRoleMap: Record<GlobalRole, PermissionRole> = {
  admin: "global-admin",
  maintainer: "global-maintainer",
  observer: "global-observer",
  observer_plus: "global-observer-plus",
  gitops: "global-gitops",
};
const ApiToClientTeamRoleMap: Record<TeamRole, PermissionRole> = {
  admin: "team-admin",
  maintainer: "team-maintainer",
  observer: "team-observer",
  observer_plus: "team-observer-plus",
  gitops: "team-gitops",
};

// This is a mapping of the application actions to the roles that have
// permission to perform them.
export const permissions = {
  "hosts.create": [
    "global-admin",
    "global-maintainer",
    "team-admin",
    "team-maintainer",
  ],
};

export type Permission = keyof typeof permissions;

const getCurrentTeamRole = (
  currentUserTeams?: ITeam[],
  currentTeamId?: number
) => {
  if (!currentUserTeams || !currentTeamId) {
    return undefined;
  }

  const currentTeam = currentUserTeams.find(
    (team) => team.id === currentTeamId
  );

  return currentTeam?.role;
};

export const usePermissions = () => {
  const { currentUser, currentTeam } = useContext(AppContext);

  const globalRole = currentUser?.global_role;
  const currentTeamRole = getCurrentTeamRole(
    currentUser?.teams,
    currentTeam?.id
  );

  const hasGlobalPermission = (permissionName: Permission) => {
    return (
      globalRole &&
      permissions[permissionName].includes(ApiToClientGlobalRoleMap[globalRole])
    );
  };

  const hasTeamPermission = (permissionName: Permission) => {
    return (
      currentTeamRole &&
      permissions[permissionName].includes(
        ApiToClientTeamRoleMap[currentTeamRole]
      )
    );
  };

  const hasPermission = (permissionName: Permission) => {
    if (
      hasGlobalPermission(permissionName) ||
      hasTeamPermission(permissionName)
    ) {
      console.log("has permission");
      console.log("global Role", globalRole);
      console.log("currentTeamRole", currentTeamRole);
    }

    return false;
  };

  return { hasPermission };
};

// };

// const hasPermission = (scope, team?) => {};

// API
// const canEditQuery = hasPermission("quieries.edit");

/**
 * Test APIS

  const { hasPermission } = usePermissions(permissionConfig);

  // obj returned
  const userPermissions = permissions(currentUser);
  if (userPermissions.canEnrollHosts) {
  }

  // obj returned
  const userPermissions = permissions(currentUser);
  if (userPermissions.enrolls.hosts) {
  }

  const { canEnrollHosts } = usePermissions(permissionConfig, currentUser);

  // boolean returned
  const canEnrollHosts = hasPermission("hosts.edit");

 */
