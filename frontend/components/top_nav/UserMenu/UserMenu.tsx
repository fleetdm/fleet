import React from "react";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { getSortedTeamOptions } from "fleet/helpers";

import PATHS from "router/paths";

// @ts-ignore
import DropdownButton from "components/buttons/DropdownButton";
import Avatar from "../../Avatar";

const baseClass = "user-menu";

interface IUserMenuProps {
  onLogout: () => void;
  onNavItemClick: (path: string) => void;
  isAnyTeamAdmin: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  currentUser: IUser;
}

const UserMenu = ({
  onLogout,
  onNavItemClick,
  isAnyTeamAdmin,
  isGlobalAdmin,
  currentUser,
}: IUserMenuProps): JSX.Element => {
  const accountNavigate = onNavItemClick(PATHS.USER_SETTINGS);
  const dropdownItems = [
    {
      label: "My account",
      onClick: accountNavigate,
    },
    {
      label: "Documentation",
      onClick: () => window.open("https://fleetdm.com/docs", "_blank"),
    },
    {
      label: "Sign out",
      onClick: onLogout,
    },
  ];

  if (isGlobalAdmin) {
    const manageUsersNavigate = onNavItemClick(PATHS.ADMIN_USERS);

    const manageUserNavItem = {
      label: "Manage users",
      onClick: manageUsersNavigate,
    };
    dropdownItems.unshift(manageUserNavItem);
  }

  if (currentUser && (isAnyTeamAdmin || isGlobalAdmin)) {
    const userAdminTeams = currentUser.teams.filter(
      (thisTeam: ITeam) => thisTeam.role === "admin"
    );
    const sortedTeams = getSortedTeamOptions(userAdminTeams);
    const settingsPath =
      currentUser.global_role === "admin"
        ? PATHS.ADMIN_SETTINGS
        : `${PATHS.ADMIN_TEAMS}/${sortedTeams[0].value}/members`;
    const settingsNavigate = onNavItemClick(settingsPath);
    const adminNavItem = {
      label: "Settings",
      onClick: settingsNavigate,
    };
    dropdownItems.unshift(adminNavItem);
  }

  return (
    <div className={baseClass}>
      <DropdownButton options={dropdownItems}>
        <Avatar
          className={`${baseClass}__avatar-image`}
          user={{ gravatarURL: currentUser.gravatarURL }}
          size="small"
        />
      </DropdownButton>
    </div>
  );
};
export default UserMenu;
