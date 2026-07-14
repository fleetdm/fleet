import React, { useContext, useMemo, useRef, useState } from "react";
import Select, {
  components,
  DropdownIndicatorProps,
  GroupBase,
  MenuListProps,
  OptionProps,
  SelectInstance,
  StylesConfig,
} from "react-select-5";
import { browserHistory } from "react-router";
import classnames from "classnames";

import { COLORS } from "styles/var/colors";
import { PADDING } from "styles/var/padding";
import { FONT_SIZES, FONT_WEIGHTS } from "styles/var/fonts";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import { IDropdownOption } from "interfaces/dropdownOption";
import {
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  ITeamSummary,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";

import Icon from "components/Icon";

// Augment react-select's selectProps so we can pass search + focus handlers
// down into the custom MenuList component.
declare module "react-select-5/dist/declarations/src/Select" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>
  > {
    searchQuery?: string;
    onChangeSearchQuery?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    onClickAddTeam?: () => void;
    showAddTeamButton?: boolean;
  }
}

export interface INumberDropdownOption extends Omit<IDropdownOption, "value"> {
  value: number; // Redefine the value property to be just number
}

const generateDropdownOptions = (
  teams: ITeamSummary[] | undefined,
  includeAllTeams: boolean,
  includeNoTeams?: boolean
): INumberDropdownOption[] => {
  if (!teams) {
    return [];
  }

  const options: INumberDropdownOption[] = teams.map((team) => ({
    disabled: false,
    label: team.name,
    value: team.id,
  }));

  const filtered = options.filter(
    (o) =>
      !(
        (o.label === APP_CONTEXT_NO_TEAM_SUMMARY.name && !includeNoTeams) ||
        (o.label === APP_CONTEXT_ALL_TEAMS_SUMMARY.name && !includeAllTeams)
      )
  );

  return filtered;
};

const filterOptionsBySearch = (
  options: INumberDropdownOption[],
  searchQuery: string
) => {
  const query = searchQuery.toLowerCase().trim();
  if (query === "") {
    return options;
  }
  return options.filter((option) => {
    if (typeof option.label !== "string") return false;
    return option.label.toLowerCase().includes(query);
  });
};

const getOptionBackgroundColor = (
  state: OptionProps<
    INumberDropdownOption,
    false,
    GroupBase<INumberDropdownOption>
  >
) => {
  return state.isFocused ? COLORS["ui-fleet-black-5"] : "transparent";
};

interface ITeamsDropdownProps {
  currentUserTeams: ITeamSummary[];
  selectedTeamId?: number;
  includeAllTeams?: boolean;
  includeNoTeams?: boolean;
  isDisabled?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
  /** Indicates that this teams dropdown should be styled as a form field */
  asFormField?: boolean;
}

const baseClass = "team-dropdown";

type ICustomMenuListProps = MenuListProps<INumberDropdownOption, false>;

const CustomMenuList = (props: ICustomMenuListProps) => {
  const { selectProps, children } = props;
  const {
    searchQuery,
    onChangeSearchQuery,
    onClickAddTeam,
    showAddTeamButton,
  } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  return (
    <components.MenuList {...props}>
      <div className={`${baseClass}__search-row`}>
        <div className={`${baseClass}__search-field`}>
          <Icon name="search" />
          <input
            ref={inputRef}
            className={`${baseClass}__search-input`}
            type="text"
            name="team-search-input"
            placeholder="Search fleets"
            value={searchQuery ?? ""}
            // Prevent the mousedown from moving focus to the input via the
            // browser's default — that would blur react-select's internal
            // input and close the menu. We focus the input ourselves so the
            // user can still type.
            onMouseDown={(e) => {
              e.preventDefault();
              e.stopPropagation();
              inputRef.current?.focus();
            }}
            onClick={(e) => e.stopPropagation()}
            // Stop keydowns from bubbling to react-select — otherwise Space,
            // arrows, Enter would trigger option selection instead of typing.
            onKeyDown={(e) => e.stopPropagation()}
            onChange={onChangeSearchQuery}
          />
        </div>
        {showAddTeamButton && (
          <button
            type="button"
            className={`${baseClass}__add-team-button`}
            aria-label="Add fleet"
            onMouseDown={(e) => {
              // Prevent focus transfer / react-select blur so the click
              // handler fires on a menu that's still mounted.
              e.preventDefault();
              e.stopPropagation();
            }}
            onClick={(e) => {
              e.stopPropagation();
              onClickAddTeam?.();
            }}
          >
            <Icon name="plus" />
          </button>
        )}
      </div>
      {children}
    </components.MenuList>
  );
};

const TeamsDropdown = ({
  currentUserTeams,
  selectedTeamId,
  includeAllTeams = true,
  includeNoTeams = false,
  isDisabled = false,
  onChange,
  onOpen,
  onClose,
  asFormField = false,
}: ITeamsDropdownProps): JSX.Element => {
  const { isGlobalAdmin } = useContext(AppContext);
  const [searchQuery, setSearchQuery] = useState("");
  const selectRef = useRef<SelectInstance<INumberDropdownOption, false>>(null);

  const teamOptions: INumberDropdownOption[] = useMemo(
    () =>
      generateDropdownOptions(
        currentUserTeams,
        includeAllTeams,
        includeNoTeams
      ),
    [currentUserTeams, includeAllTeams, includeNoTeams]
  );

  const filteredOptions = useMemo(
    () => filterOptionsBySearch(teamOptions, searchQuery),
    [teamOptions, searchQuery]
  );

  const selectedValue = teamOptions.find(
    (option) => selectedTeamId === option.value
  )
    ? selectedTeamId
    : teamOptions[0]?.value;

  const dropdownWrapperClasses = classnames(`${baseClass}-wrapper`, {
    disabled: isDisabled || undefined,
  });

  const handleMenuOpen = () => {
    onOpen?.();
  };

  const handleMenuClose = () => {
    setSearchQuery("");
    onClose?.();
  };

  const onChangeSearchQuery = (event: React.ChangeEvent<HTMLInputElement>) => {
    event.stopPropagation();
    setSearchQuery(event.target.value);
  };

  const onClickAddTeam = () => {
    selectRef.current?.blur();
    browserHistory.push(PATHS.ADMIN_FLEETS);
  };

  const CustomDropdownIndicator = (
    props: DropdownIndicatorProps<
      INumberDropdownOption,
      false,
      GroupBase<INumberDropdownOption>
    >
  ) => {
    const { isFocused, selectProps: dropdownSelectProps } = props;
    const color =
      isFocused || dropdownSelectProps.menuIsOpen
        ? "core-fleet-black"
        : "ui-fleet-black-75";

    return (
      <components.DropdownIndicator {...props} className={baseClass}>
        <Icon
          name="chevron-down"
          color={color}
          className={`${baseClass}__icon`}
        />
      </components.DropdownIndicator>
    );
  };

  const [variableControlStyles, variableSingleValueStyles] = asFormField
    ? [
        {
          padding: ".5rem 1rem",
          backgroundColor: COLORS["ui-light-grey"],
        },
        {},
      ]
    : [
        {
          padding: "8px 0",
          backgroundColor: "initial",
          border: 0,
        },
        {
          fontSize: "24px",
        },
      ];

  // see https://react-select.com/styles#the-styles-prop
  const customStyles: StylesConfig<INumberDropdownOption, false> = {
    control: (baseStyles, state) => ({
      ...baseStyles,
      ...variableControlStyles,
      display: "flex",
      flexDirection: "row",
      borderRadius: "4px",
      boxShadow: "none",
      cursor: "pointer",
      "&:hover": {
        boxShadow: "none",
        ".team-dropdown__single-value": {
          color: COLORS["core-fleet-black"],
        },
        ".team-dropdown__indicator path": {
          stroke: COLORS["ui-fleet-black-75-over"],
        },
      },
      // When tabbing
      // Relies on --is-focused for styling as &:focus-visible cannot be applied
      "&.team-dropdown__control--is-focused": {
        ".team-dropdown__indicator path": {
          stroke: COLORS["ui-fleet-black-75-over"],
        },
      },
      ...(state.isDisabled && {
        ".team-dropdown__single-value": {
          color: COLORS["ui-fleet-black-50"],
        },
        ".team-dropdown__indicator path": {
          stroke: COLORS["ui-fleet-black-50"],
        },
      }),
      // When clicking
      "&:active": {
        ".team-dropdown__single-value": {
          color: COLORS["ui-fleet-black-75-down"],
        },
        ".team-dropdown__indicator path": {
          stroke: COLORS["ui-fleet-black-75-down"],
        },
      },
      ...(state.menuIsOpen && {
        ".team-dropdown__indicator svg": {
          transform: "rotate(180deg)",
          transition: "transform 0.25s ease",
        },
      }),
    }),
    singleValue: (baseStyles) => ({
      ...baseStyles,
      ...variableSingleValueStyles,
      color: COLORS["core-fleet-black"],
      lineHeight: "normal",
      paddingLeft: 0,
      paddingRight: "8px",
      margin: 0,
      fontWeight: "600",
      // omit grid-column-end for automatic width
      gridArea: "1/1/2",
    }),
    dropdownIndicator: (baseStyles) => ({
      ...baseStyles,
      display: "flex",
      padding: "2px",
      margin: "0 5px",
      svg: {
        transition: "transform 0.25s ease",
      },
    }),
    menu: (baseStyles) => ({
      ...baseStyles,
      backgroundColor: COLORS["core-fleet-white"],
      boxShadow: `0 2px 6px rgba(0, 0, 0, 0.1), 0 0 0 1px ${COLORS["ui-fleet-black-10"]}`,
      borderRadius: "4px",
      zIndex: 6,
      overflow: "hidden",
      border: 0,
      marginTop: 0,
      minWidth: "340px",
      maxHeight: "none",
      position: "absolute",
      left: "0",
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (baseStyles) => ({
      ...baseStyles,
      // Search row provides the top padding via its own margin so options
      // scroll under a sticky-feeling search area with no gap.
      padding: PADDING["pad-small"],
      paddingTop: 0,
      ".team-dropdown__menu-notice--no-options": {
        textAlign: "left",
        color: COLORS["ui-fleet-black-50"],
        fontSize: FONT_SIZES["xx-small"],
        fontWeight: FONT_WEIGHTS.regular,
        padding: "10px 8px",
      },
    }),
    valueContainer: (baseStyles) => ({
      ...baseStyles,
      padding: 0,
    }),
    input: (baseStyles) => ({
      ...baseStyles,
      overflow: "hidden",
      textOverflow: "ellipsis",
      whiteSpace: "nowrap",
      margin: 0,
      color: COLORS["core-fleet-black"],
    }),
    option: (baseStyles, state) => ({
      ...baseStyles,
      padding: "10px 8px",
      fontSize: "13px",
      borderRadius: "4px",
      backgroundColor: getOptionBackgroundColor(state),
      fontWeight: state.isSelected ? "600" : "normal",
      color: COLORS["core-fleet-black"],
      "&:hover": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-fleet-black-5"],
      },
      "&:active": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-fleet-black-5"],
      },
      ...(state.isDisabled && {
        color: COLORS["ui-fleet-black-50"],
        fontStyle: "italic",
      }),
    }),
  };

  return (
    <div className={dropdownWrapperClasses}>
      <Select<INumberDropdownOption, false>
        ref={selectRef}
        options={filteredOptions}
        placeholder="All fleets"
        onChange={(newValue) => {
          if (newValue) {
            onChange(newValue.value);
          }
        }}
        isDisabled={isDisabled}
        // Native react-select search is disabled — we use our own visible
        // search input rendered by CustomMenuList.
        isSearchable={false}
        onMenuOpen={handleMenuOpen}
        onMenuClose={handleMenuClose}
        noOptionsMessage={() => "No matching fleets"}
        styles={customStyles}
        components={{
          DropdownIndicator: CustomDropdownIndicator,
          IndicatorSeparator: () => null,
          MenuList: CustomMenuList,
        }}
        value={teamOptions.find((option) => option.value === selectedValue)}
        isOptionSelected={() => false} // Hides any styling on selected option
        className={baseClass}
        classNamePrefix={baseClass}
        searchQuery={searchQuery}
        onChangeSearchQuery={onChangeSearchQuery}
        onClickAddTeam={onClickAddTeam}
        showAddTeamButton={!!isGlobalAdmin}
      />
    </div>
  );
};

export default TeamsDropdown;
