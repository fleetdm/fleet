import React, { useEffect, useState } from "react";
import { keyframes } from "@emotion/react";
import Select, {
  StylesConfig,
  DropdownIndicatorProps,
  OptionProps,
  components,
  GroupBase,
} from "react-select-5";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";
import { getSortedTeamOptions } from "utilities/helpers";

import { PADDING } from "styles/var/padding";
import { COLORS } from "styles/var/colors";

import Icon from "components/Icon";
import AvatarTopNav from "../../AvatarTopNav";

const baseClass = "user-menu";

interface IUserMenuProps {
  onLogout: () => void;
  onUserMenuItemClick: (path: string) => void;
  isAnyTeamAdmin: boolean | undefined;
  isGlobalAdmin: boolean | undefined;
  currentUser: IUser;
}

const bounceDownAnimation = keyframes`
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(3px);
  }
`;

const getOptionBackgroundColor = (state: any) => {
  return state.isFocused ? COLORS["ui-vibrant-blue-10"] : "transparent";
};

const CustomDropdownIndicator = (
  props: DropdownIndicatorProps<
    IDropdownOption,
    false,
    GroupBase<IDropdownOption>
  >
) => {
  return (
    <components.DropdownIndicator {...props} className={baseClass}>
      <Icon
        name="chevron-down"
        color="core-fleet-white"
        className={`${baseClass}__icon`}
        size="small"
      />
    </components.DropdownIndicator>
  );
};

const CustomOption: React.FC<OptionProps<IDropdownOption, false>> = (props) => {
  const { innerRef, data } = props;

  return (
    <components.Option {...props} isFocused={false}>
      <div
        className={`${baseClass}__option`}
        ref={innerRef}
        // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
        tabIndex={0}
        role="menuitem"
      >
        {data.label}
      </div>
    </components.Option>
  );
};

const UserMenu = ({
  onLogout,
  onUserMenuItemClick,
  isAnyTeamAdmin,
  isGlobalAdmin,
  currentUser,
}: IUserMenuProps): JSX.Element => {
  // Work around for react-select-5 not having :focus-visible pseudo class that can style dropdown on tabbing only
  const [isKeyboardFocus, setIsKeyboardFocus] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Tab") {
        setIsKeyboardFocus(true);
      }
    };

    const handleMouseDown = () => {
      setIsKeyboardFocus(false);
    };

    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("mousedown", handleMouseDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("mousedown", handleMouseDown);
    };
  }, []);

  const dropdownItems = [
    {
      label: "My account",
      value: "my-account",
      onClick: () => onUserMenuItemClick(PATHS.ACCOUNT),
    },
    {
      label: "Documentation",
      value: "documentation",
      onClick: () => {
        window.open("https://fleetdm.com/docs", "_blank");
      },
    },
    {
      label: "Sign out",
      value: "sign-out",
      onClick: onLogout,
    },
  ];

  if (isGlobalAdmin) {
    const manageUserNavItem = {
      label: "Manage users",
      value: "manage-users",
      onClick: () => onUserMenuItemClick(PATHS.ADMIN_USERS),
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
    const adminNavItem = {
      label: "Settings",
      value: "settings",
      onClick: () => onUserMenuItemClick(settingsPath),
    };
    dropdownItems.unshift(adminNavItem);
  }

  const customStyles: StylesConfig<IDropdownOption, false> = {
    control: (provided, state) => ({
      ...provided,
      display: "flex",
      flexDirection: "row",
      width: "max-content",
      padding: "8px",
      marginRight: "8px",
      backgroundColor: "initial",
      border: "2px solid transparent", // So tabbing doesn't shift dropdown
      borderRadius: "6px",
      boxShadow: "none",
      cursor: "pointer",
      "&:hover": {
        boxShadow: "none",
        ".user-menu-select__indicator svg": {
          animation: `${bounceDownAnimation} 0.3s ease-in-out`,
        },
      },
      ...(state.isFocused &&
        isKeyboardFocus && {
          border: `2px solid ${COLORS["ui-blue-25"]}`,
          // Add other focus styles as needed
        }),
      ...(state.menuIsOpen && {
        ".user-menu-select__indicator svg": {
          transform: "rotate(180deg)",
        },
      }),
    }),
    dropdownIndicator: (provided) => ({
      ...provided,
      display: "flex",
      padding: "6px",
      svg: {
        transition: "transform 0.25s ease",
      },
    }),
    menu: (provided) => ({
      ...provided,
      boxShadow: "0 2px 6px rgba(0, 0, 0, 0.1)",
      borderRadius: "4px",
      zIndex: 6,
      marginTop: "7px",
      marginRight: "8px",
      width: "auto",
      minWidth: "100%",
      position: "absolute",
      left: "auto",
      right: "0",
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (provided) => ({
      ...provided,
      padding: PADDING["pad-small"],
      maxHeight: "initial", // Override react-select default height of 300px
    }),
    valueContainer: (provided) => ({
      ...provided,
      padding: 0,
    }),
    option: (provided, state) => ({
      ...provided,
      padding: "10px 8px",
      fontSize: "15px",
      backgroundColor: getOptionBackgroundColor(state),
      color: COLORS["tooltip-bg"], // TODO: Why the mismatch in names in colors.scss and colors.ts
      whiteSpace: "nowrap",
      "&:hover": {
        backgroundColor: COLORS["ui-vibrant-blue-10"],
      },
      "&:active": {
        backgroundColor: COLORS["ui-vibrant-blue-10"],
      },
      "&:last-child, &:nth-last-of-type(2)": {
        borderTop: `1px solid ${COLORS["ui-fleet-black-10"]}`,
      },
    }),
  };

  const renderPlaceholder = () => {
    return (
      <AvatarTopNav
        className={`${baseClass}__avatar-image`}
        user={{ gravatar_url_dark: currentUser.gravatar_url_dark }}
        size="small"
      />
    );
  };

  return (
    <div className={baseClass} data-testid="user-menu">
      <Select<IDropdownOption, false>
        options={dropdownItems}
        placeholder={renderPlaceholder()}
        styles={customStyles}
        components={{
          DropdownIndicator: CustomDropdownIndicator,
          IndicatorSeparator: () => null,
          Option: CustomOption,
          SingleValue: () => null,
        }}
        controlShouldRenderValue={false}
        isOptionSelected={() => false}
        className={baseClass}
        classNamePrefix={`${baseClass}-select`}
        menuPlacement="bottom"
        onChange={(option) => {
          option?.onClick && option?.onClick();
        }}
        isSearchable={false}
      />
    </div>
  );
};

export default UserMenu;
