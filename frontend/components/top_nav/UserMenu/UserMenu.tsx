import React from "react";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { getSortedTeamOptions } from "utilities/helpers";

import PATHS from "router/paths";

// @ts-ignore
import DropdownButton from "components/buttons/DropdownButton";
import AvatarTopNav from "../../AvatarTopNav";

const baseClass = "user-menu";

interface IUserMenuProps {
  onLogout: () => void;
  onNavItemClick: (path: string) => void;
  isAnyTeamAdmin: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  currentUser: IUser;
  isSandboxMode?: boolean;
}

const UserMenu = ({
  onLogout,
  onNavItemClick,
  isAnyTeamAdmin,
  isGlobalAdmin,
  currentUser,
  isSandboxMode = false,
}: IUserMenuProps): JSX.Element => {
  const accountNavigate = onNavItemClick(PATHS.ACCOUNT);
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

  if (isGlobalAdmin && !isSandboxMode) {
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
        ? PATHS.ADMIN_ORGANIZATION
        : `${PATHS.TEAM_DETAILS_USERS(sortedTeams[0].value)}`;
    const settingsNavigate = onNavItemClick(settingsPath);
    const adminNavItem = {
      label: "Settings",
      onClick: settingsNavigate,
    };
    dropdownItems.unshift(adminNavItem);
  }

  return (
    <div className={baseClass} data-testid="user-menu">
      <DropdownButton options={dropdownItems}>
        <AvatarTopNav
          className={`${baseClass}__avatar-image`}
          user={{ gravatar_url_dark: currentUser.gravatar_url_dark }}
          size="small"
        />
      </DropdownButton>
    </div>
  );
};
export default UserMenu;
