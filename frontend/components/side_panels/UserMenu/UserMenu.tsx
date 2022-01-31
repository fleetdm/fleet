import React, { useState } from "react";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { getSortedTeamOptions } from "fleet/helpers";
import URL_PREFIX from "router/url_prefix";

import PATHS from "router/paths";

// @ts-ignore
import DropdownButton from "components/buttons/DropdownButton";
import Avatar from "../../Avatar";

const baseClass = "user-menu";

interface IUserMenuProps {
  onLogout: () => any;
  onNavItemClick: any;
  // user: {
  //   gravatarURL?: string | undefined;
  //   name?: string;
  //   email: string;
  //   position?: string;
  // };
  user: any;
  isAnyTeamAdmin: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  currentUser: IUser | null;
}

const UserMenu = ({
  onLogout,
  onNavItemClick,
  user,
  isAnyTeamAdmin,
  isGlobalAdmin,
  currentUser,
}: IUserMenuProps): JSX.Element => {
  // const [isOpened, setIsOpened] = useState<boolean>(false);

  console.log("user", user);
  const settingsNavigate = onNavItemClick(PATHS.ADMIN_SETTINGS);
  const manageUsersNavigate = onNavItemClick(PATHS.ADMIN_USERS);
  const accountNavigate = onNavItemClick(PATHS.USER_SETTINGS);

  const dropdownItems = [
    {
      label: "My account",
      onClick: accountNavigate,
    },
    {
      label: "Documentation",
      onClick: () =>
        window.open(
          "https://github.com/fleetdm/fleet/blob/main/docs/README.md",
          "_blank"
        ),
    },
    {
      label: "Sign out",
      onClick: onLogout,
    },
  ];

  console.log("isGlobalAdmin", isGlobalAdmin);
  if (currentUser && isGlobalAdmin) {
    const manageUserNavItem = {
      label: "Manage users",
      onClick: manageUsersNavigate,
    };
    dropdownItems.unshift(manageUserNavItem);
  }

  // TODO: Fix reroute for team admin!
  if (currentUser && (isAnyTeamAdmin || isGlobalAdmin)) {
    const userAdminTeams = currentUser.teams.filter(
      (thisTeam: ITeam) => thisTeam.role === "admin"
    );
    const sortedTeams = getSortedTeamOptions(userAdminTeams);
    const adminNavItem = {
      label: "Settings",
      onClick: settingsNavigate,
    };
    //   [
    //   {
    //     icon: "settings",
    //     name: "Settings",
    //     iconName: "settings",
    //     location: {
    //       regex: new RegExp(`^${URL_PREFIX}/settings/`),
    //       pathname:
    //         currentUser.global_role === "admin"
    //           ? PATHS.ADMIN_SETTINGS
    //           : `${PATHS.ADMIN_TEAMS}/${sortedTeams[0].value}/members`,
    //     },
    //   },
    // ];
    dropdownItems.unshift(adminNavItem);
  }

  return (
    <div className={baseClass}>
      <DropdownButton options={dropdownItems}>
        <Avatar
          className={`${baseClass}__avatar-image`}
          user={user}
          size="small"
        />
      </DropdownButton>
    </div>
  );
};
export default UserMenu;
