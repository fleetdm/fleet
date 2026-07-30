import React, { useContext, useEffect, useState } from "react";
import Select, {
  components,
  DropdownIndicatorProps,
  GroupBase,
  OptionProps,
  StylesConfig,
} from "react-select-5";

import { IUser } from "interfaces/user";
import { ITeamSummary } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";
import permissions from "utilities/permissions";
import { AppContext } from "context/app";

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
  currentTeam: ITeamSummary | undefined;
}

const getOptionBackgroundColor = (
  state: OptionProps<IDropdownOption, false, GroupBase<IDropdownOption>>
) => {
  return state.isFocused ? COLORS["ui-fleet-black-10"] : "transparent";
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
        color="ui-fleet-black-75"
        className={`${baseClass}__icon`}
        size="small"
      />
    </components.DropdownIndicator>
  );
};

const CustomOption: React.FC<
  OptionProps<IDropdownOption, false> & { isKeyboardFocus: boolean }
> = (props) => {
  const { innerRef, data, isFocused, isKeyboardFocus } = props;

  return (
    <>
      {data.hasDividerBefore && (
        <div
          className={`${baseClass}__divider`}
          aria-hidden="true"
          role="presentation"
        />
      )}
      <components.Option
        {...props}
        isFocused={isKeyboardFocus ? isFocused : false} // work around to not preselect first option unless keyboarding
      >
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
    </>
  );
};

const UserMenu = ({
  onLogout,
  onUserMenuItemClick,
  isAnyTeamAdmin,
  isGlobalAdmin,
  currentUser,
  currentTeam,
}: IUserMenuProps): JSX.Element => {
  const { availableTeams, isPremiumTier, isSandboxMode } = useContext(
    AppContext
  );

  // Work around for react-select-5 not having :focus-visible pseudo class that can style dropdown on keyboard tab only
  // Work around preventing react-select-5 from auto focusing first option unless using keyboard
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

  const dropdownItems: IDropdownOption[] = [
    {
      label: "Labels",
      value: "labels",
      onClick: () => onUserMenuItemClick(PATHS.MANAGE_LABELS),
    },
  ];

  if (isGlobalAdmin) {
    if (!isSandboxMode) {
      dropdownItems.push({
        label: "Organization settings",
        value: "organization-settings",
        hasDividerBefore: true,
        onClick: () => onUserMenuItemClick(PATHS.ADMIN_ORGANIZATION),
      });
    } else {
      dropdownItems.push({
        label: "Integrations",
        value: "integrations",
        hasDividerBefore: true,
        onClick: () => onUserMenuItemClick(PATHS.ADMIN_INTEGRATIONS),
      });
    }

    if (!isSandboxMode) {
      dropdownItems.push({
        label: "Integrations",
        value: "integrations",
        onClick: () => onUserMenuItemClick(PATHS.ADMIN_INTEGRATIONS),
      });
      dropdownItems.push({
        label: "Users",
        value: "users",
        onClick: () => onUserMenuItemClick(PATHS.ADMIN_USERS),
      });
    }

    if (isPremiumTier) {
      dropdownItems.push({
        label: "Fleets",
        value: "fleets",
        onClick: () => onUserMenuItemClick(PATHS.ADMIN_FLEETS),
      });
    }
  } else if (currentUser && isAnyTeamAdmin) {
    // Resolved at click time so availableTeams is guaranteed to be loaded.
    const getTargetTeamId = () => {
      const currentTeamIsAdmin =
        currentTeam && permissions.isTeamAdmin(currentUser, currentTeam.id);
      // Use the current team if the user is an admin of it, otherwise fall back
      // to the first team (alphabetical) the user is an admin of.
      // availableTeams is pre-sorted alphabetically by AppContext.
      return currentTeamIsAdmin
        ? currentTeam.id
        : availableTeams?.find((t) =>
            permissions.isTeamAdmin(currentUser, t.id)
          )?.id;
    };

    dropdownItems.push({
      label: "Users",
      value: "team-users",
      hasDividerBefore: true,
      onClick: () =>
        onUserMenuItemClick(PATHS.FLEET_DETAILS_USERS(getTargetTeamId())),
    });
    dropdownItems.push({
      label: "Agent options",
      value: "team-agent-options",
      onClick: () =>
        onUserMenuItemClick(PATHS.FLEET_DETAILS_OPTIONS(getTargetTeamId())),
    });
    dropdownItems.push({
      label: "Settings",
      value: "team-settings",
      onClick: () =>
        onUserMenuItemClick(PATHS.FLEET_DETAILS_SETTINGS(getTargetTeamId())),
    });
  }

  dropdownItems.push({
    label: "My account",
    value: "my-account",
    hasDividerBefore: true,
    onClick: () => onUserMenuItemClick(PATHS.ACCOUNT),
  });
  dropdownItems.push({
    label: "Documentation",
    value: "documentation",
    onClick: () => {
      window.open("https://fleetdm.com/docs", "_blank");
    },
  });
  dropdownItems.push({
    label: "Sign out",
    value: "sign-out",
    hasDividerBefore: true,
    onClick: onLogout,
  });

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
      borderRadius: "3px", // Match other nav border after their focused offset
      boxShadow: "none",
      cursor: "pointer",
      "&:hover": {
        boxShadow: "none",
      },
      ...(state.isFocused &&
        isKeyboardFocus && {
          outline: `1px solid ${COLORS["core-fleet-black"]}`,
          outlineOffset: "1px",
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
      backgroundColor: COLORS["core-fleet-white"],
      boxShadow: `0 2px 6px rgba(0, 0, 0, 0.1), 0 0 0 1px ${COLORS["ui-fleet-black-10"]}`,
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
      color: COLORS["core-fleet-black"],
      whiteSpace: "nowrap",
      "&:hover": {
        backgroundColor: COLORS["ui-fleet-black-5"],
      },
    }),
  };

  const renderPlaceholder = () => {
    return (
      <AvatarTopNav
        className={`${baseClass}__avatar-image`}
        user={{ gravatar_url: currentUser.gravatar_url }}
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
          Option: (props) => (
            <CustomOption {...props} isKeyboardFocus={isKeyboardFocus} />
          ),
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
