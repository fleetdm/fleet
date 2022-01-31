import React, { useState } from "react";

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
}

const UserMenu = ({
  onLogout,
  onNavItemClick,
  user,
}: IUserMenuProps): JSX.Element => {
  // const [isOpened, setIsOpened] = useState<boolean>(false);

  console.log("user", user);
  const settingsNavigate = onNavItemClick(PATHS.ADMIN_SETTINGS);
  const manageUsersNavigate = onNavItemClick(PATHS.ADMIN_USERS);
  const accountNavigate = onNavItemClick(PATHS.USER_SETTINGS);

  const dropdownItems = [
    {
      label: "Settings",
      onClick: settingsNavigate,
    },
    {
      label: "Manage users",
      onClick: manageUsersNavigate,
    },
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
