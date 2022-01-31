import React, { useState } from "react";

import PATHS from "router/paths";

// @ts-ignore
import DropdownButton from "components/buttons/DropdownButton";
import Avatar from "../../Avatar";

const baseClass = "user-menu";

interface IUserMenuProps {
  onLogout: () => any;
  onNavItemClick: any;
  user: {
    gravatarURL?: string | undefined;
    name?: string;
    email: string;
    position?: string;
  };
}

const UserMenu = ({
  onLogout,
  onNavItemClick,
  user,
}: IUserMenuProps): JSX.Element => {
  // const [isOpened, setIsOpened] = useState<boolean>(false);

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
