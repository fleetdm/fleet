import { AppContext } from "context/app";
import { useContext } from "react";

export type Roles =
  | "global-admin"
  | "global-maintainer"
  | "global-observer"
  | "global-observer-plus"
  | "team-admin"
  | "team-maintainer"
  | "team-observer"
  | "team-observer-plus"
  | "current-team-admin"
  | "current-team-maintainer"
  | "current-team-observer"
  | "current-team-plus"
  | "gitops";

export const permissions = {
  "hosts.create": [
    "global-admin",
    "global-maintainer",
    "team-admin",
    "team-maintainer",
  ],
  "query.edit": ["global-admin", "global-maintainer"],
  "query.run": [
    "global-admin",
    "global-maintainer",
    "global-observer-plus",
    "current-team-admin",
    "current-team-maintainer",
    "current-team-observer-plus",
  ],
};

export type Permission = keyof typeof permissions;

export const usePermissions = () => {
  const { currentUser } = useContext(AppContext)

  const hasPermission = (permissionName: Permission, teamId?: number) => {

    // check global role
    const userRole = getUserRole(teamId);

    // if global role check role vs permission
    if (permissions[permissionName].includes(userRole)) {
      console.log("has permission", useRole);
    }

    // if teamId provided get role for team

    // check role for team vs permission

    const permissionRoles = permissions[permissionName];

    if (permissionRoles.includes(userRole)) {
      return true;
    }

    if (teamId) {
      const teamRole = getTeamRole(teamId);

      if (permissionRoles.includes(teamRole)) {
        return true;
      }
    }

    return false;
  };

  const hasPermissionTeam = () => {

  }

//   return { hasPermission };
// };

// const hasPermission = (scope, team?) => {};

// API
// const canEditQuery = hasPermission("quieries.edit");
