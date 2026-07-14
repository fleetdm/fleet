import React, { useContext, useEffect, useMemo, useRef, useState } from "react";
import Select, {
  components,
  GroupBase,
  MenuListProps,
  SelectInstance,
  StylesConfig,
} from "react-select-5";
import { browserHistory } from "react-router";
import classnames from "classnames";

import { COLORS } from "styles/var/colors";
import { PADDING } from "styles/var/padding";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import { IDropdownOption } from "interfaces/dropdownOption";
import {
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  ITeamSummary,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

declare module "react-select-5/dist/declarations/src/Select" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>
  > {
    searchQuery?: string;
    onChangeSearchQuery?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    // Forwards navigation keys (Arrow/Enter/Escape) from the in-menu search
    // input to react-select's own (hidden) input so option highlighting and
    // selection still work while the search input has focus.
    forwardNavKey?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
    onClickAddTeam?: () => void;
    showAddTeamButton?: boolean;
    showSearch?: boolean;
  }
}

// Search input only appears once the list is long enough to be worth
// filtering. Below this, the "+" button (when the user can add fleets) still
// renders on its own row.
const MIN_TEAMS_FOR_SEARCH = 6;

export interface INumberDropdownOption extends Omit<IDropdownOption, "value"> {
  value: number;
}

const generateDropdownOptions = (
  teams: ITeamSummary[] | undefined,
  includeAllTeams: boolean,
  includeNoTeams?: boolean
): INumberDropdownOption[] => {
  if (!teams) return [];

  const options: INumberDropdownOption[] = teams.map((team) => ({
    disabled: false,
    label: team.name,
    value: team.id,
  }));

  return options.filter(
    (o) =>
      !(
        (o.label === APP_CONTEXT_NO_TEAM_SUMMARY.name && !includeNoTeams) ||
        (o.label === APP_CONTEXT_ALL_TEAMS_SUMMARY.name && !includeAllTeams)
      )
  );
};

const filterOptionsBySearch = (
  options: INumberDropdownOption[],
  searchQuery: string
) => {
  const query = searchQuery.toLowerCase().trim();
  if (query === "") return options;
  return options.filter((option) => {
    if (typeof option.label !== "string") return false;
    return option.label.toLowerCase().includes(query);
  });
};

const NAV_KEYS = new Set(["ArrowDown", "ArrowUp", "Enter", "Escape"]);

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

const CustomMenuList = (props: MenuListProps<INumberDropdownOption, false>) => {
  const { selectProps } = props;
  const {
    searchQuery,
    onChangeSearchQuery,
    forwardNavKey,
    onClickAddTeam,
    showAddTeamButton,
    showSearch,
  } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Autofocus the search input when the menu opens (only if it's rendered).
  useEffect(() => {
    if (showSearch) inputRef.current?.focus();
  }, [showSearch]);

  const handleInputClick = (
    event: React.MouseEvent<HTMLInputElement, MouseEvent>
  ) => {
    inputRef.current?.focus();
    event.stopPropagation();
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (NAV_KEYS.has(event.key)) {
      // Let react-select's own keyDown handler do option highlighting /
      // selection / menu close, but keep visible focus on the search input.
      event.preventDefault();
      forwardNavKey?.(event);
      return;
    }
    event.stopPropagation();
  };

  // Block clicks on the MenuList's own padding from bubbling to the Menu
  // element's onMouseDown handler in react-select, which would otherwise call
  // focusInput() and steal focus to the hidden Control input — making a
  // subsequent click on the search input look like a close.
  // When there are few enough fleets that no search is needed, the "+" moves
  // to the bottom as a full-width labeled action instead of a top-row icon.
  const addTeamAtTop = showAddTeamButton && showSearch;
  const addTeamAtBottom = showAddTeamButton && !showSearch;

  const addTeamMouseDown = (event: React.MouseEvent) => {
    // Keep focus out of react-select's hidden input so the click fires on a
    // still-mounted menu.
    event.preventDefault();
    event.stopPropagation();
  };

  const addTeamClick = (event: React.MouseEvent) => {
    event.stopPropagation();
    onClickAddTeam?.();
  };

  return (
    <components.MenuList
      {...props}
      innerProps={{
        ...props.innerProps,
        onMouseDown: (event: React.MouseEvent) => event.stopPropagation(),
      }}
    >
      {(showSearch || addTeamAtTop) && (
        <div className={`${baseClass}__search-row`}>
          {showSearch && (
            <div className={`${baseClass}__search-field`}>
              <input
                ref={inputRef}
                className={`${baseClass}__search-input`}
                value={searchQuery ?? ""}
                name="team-search-input"
                type="text"
                placeholder="Search fleets"
                onKeyDown={handleKeyDown}
                onChange={onChangeSearchQuery}
                onClick={handleInputClick}
                onMouseDown={(event) => event.stopPropagation()}
              />
              <Icon name="search" />
            </div>
          )}
          {addTeamAtTop && (
            <button
              type="button"
              className={`${baseClass}__add-team-button`}
              aria-label="Add fleet"
              onMouseDown={addTeamMouseDown}
              onClick={addTeamClick}
            >
              <Icon name="plus" />
            </button>
          )}
        </div>
      )}
      <div className={`${baseClass}__options-spacer`} />
      {props.children}
      {addTeamAtBottom && (
        <button
          type="button"
          className={`${baseClass}__add-team-footer`}
          onMouseDown={addTeamMouseDown}
          onClick={addTeamClick}
        >
          <Icon name="plus" />
          <span>Add fleet</span>
        </button>
      )}
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
  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const selectRef = useRef<SelectInstance<INumberDropdownOption, false>>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

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

  const showSearch = teamOptions.length >= MIN_TEAMS_FOR_SEARCH;

  const selectedValue = teamOptions.find(
    (option) => selectedTeamId === option.value
  )
    ? selectedTeamId
    : teamOptions[0]?.value;

  const selectedLabel =
    teamOptions.find((o) => o.value === selectedValue)?.label ?? "All fleets";

  // Close menu when clicking outside the wrapper.
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        menuIsOpen &&
        wrapperRef.current &&
        !wrapperRef.current.contains(event.target as Node)
      ) {
        setMenuIsOpen(false);
        setSearchQuery("");
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [menuIsOpen]);

  const toggleMenu = () => {
    if (isDisabled) return;
    setMenuIsOpen((open) => {
      const next = !open;
      if (next) onOpen?.();
      else {
        setSearchQuery("");
        onClose?.();
      }
      return next;
    });
  };

  const handleChange = (newValue: INumberDropdownOption | null) => {
    if (!newValue) return;
    onChange(newValue.value);
    setSearchQuery("");
    setMenuIsOpen(false);
    onClose?.();
  };

  const onChangeSearchQuery = (event: React.ChangeEvent<HTMLInputElement>) => {
    event.stopPropagation();
    setSearchQuery(event.target.value);
  };

  // Forwards a navigation key from the in-menu search input to react-select's
  // hidden input so its built-in keyDown handler runs (option highlighting,
  // selection, menu close).
  const forwardNavKey = (event: React.KeyboardEvent<HTMLInputElement>) => {
    const input = ((selectRef.current as unknown) as {
      inputRef?: HTMLInputElement | null;
    })?.inputRef;
    if (!input) return;
    input.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: event.key,
        code: event.code,
        bubbles: true,
        cancelable: true,
      })
    );
  };

  const onClickAddTeam = () => {
    setMenuIsOpen(false);
    setSearchQuery("");
    onClose?.();
    browserHistory.push(`${PATHS.ADMIN_FLEETS}?create_fleet=1`);
  };

  const wrapperClasses = classnames(`${baseClass}-wrapper`, {
    [`${baseClass}-wrapper--form-field`]: asFormField,
    [`${baseClass}-wrapper--disabled`]: isDisabled,
  });

  const buttonClasses = classnames(`${baseClass}__button`, {
    [`${baseClass}__button--form-field`]: asFormField,
  });

  const iconClasses = classnames(`${baseClass}__icon`, {
    [`${baseClass}__icon--open`]: menuIsOpen,
  });

  // Menu + option styling only — the visible trigger is a real Fleet Button
  // above, and the react-select Control is hidden but kept in the DOM so its
  // hidden input can receive dispatched keydown events for nav keys.
  const customStyles: StylesConfig<INumberDropdownOption, false> = {
    control: () => ({
      position: "absolute",
      top: 0,
      left: 0,
      width: 1,
      height: 1,
      overflow: "hidden",
      opacity: 0,
      pointerEvents: "none",
    }),
    menu: (baseStyles) => ({
      ...baseStyles,
      backgroundColor: COLORS["core-fleet-white"],
      boxShadow: `0 2px 6px rgba(0, 0, 0, 0.1), 0 0 0 1px ${COLORS["ui-fleet-black-10"]}`,
      borderRadius: "4px",
      zIndex: 6,
      overflow: "hidden",
      border: 0,
      marginTop: PADDING["pad-xsmall"],
      width: 340,
      maxHeight: "none",
      position: "absolute",
      left: 0,
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (baseStyles) => ({
      ...baseStyles,
      maxHeight: 360,
      // Search field owns the top padding via its own padding-top so options
      // scrolling up are hidden by the sticky background.
      paddingTop: 0,
      paddingBottom: PADDING["pad-small"],
      paddingLeft: PADDING["pad-small"],
      paddingRight: PADDING["pad-small"],
    }),
    noOptionsMessage: (baseStyles) => ({
      ...baseStyles,
      padding: "10px 8px",
      fontSize: "13px",
      textAlign: "left",
      color: COLORS["ui-fleet-black-75"],
    }),
    option: (baseStyles, state) => ({
      ...baseStyles,
      padding: "10px 8px",
      fontSize: "13px",
      borderRadius: "4px",
      backgroundColor: state.isFocused
        ? COLORS["ui-fleet-black-5"]
        : "transparent",
      fontWeight: state.isSelected ? 600 : "normal",
      color: COLORS["core-fleet-black"],
      cursor: "pointer",
      overflow: "hidden",
      textOverflow: "ellipsis",
      whiteSpace: "nowrap",
      "&:hover": {
        backgroundColor: COLORS["ui-fleet-black-5"],
      },
    }),
  };

  return (
    <div className={wrapperClasses} ref={wrapperRef}>
      <Button
        variant="unstyled"
        type="button"
        onClick={toggleMenu}
        disabled={isDisabled}
        className={buttonClasses}
        ariaHasPopup="listbox"
        ariaExpanded={menuIsOpen}
      >
        <span className={`${baseClass}__button-label`}>{selectedLabel}</span>
        <Icon
          name="chevron-down"
          color={menuIsOpen ? "core-fleet-black" : "ui-fleet-black-75"}
          className={iconClasses}
        />
      </Button>
      <Select<INumberDropdownOption, false>
        ref={selectRef}
        options={filteredOptions}
        value={teamOptions.find((option) => option.value === selectedValue)}
        onChange={handleChange}
        isDisabled={isDisabled}
        isSearchable={false}
        menuIsOpen={menuIsOpen}
        onMenuOpen={() => setMenuIsOpen(true)}
        onMenuClose={() => {
          setMenuIsOpen(false);
          setSearchQuery("");
        }}
        styles={customStyles}
        components={{
          MenuList: CustomMenuList,
          DropdownIndicator: () => null,
          IndicatorSeparator: () => null,
        }}
        // Hidden input is never directly user-focused; it just receives
        // dispatched keydown events from the in-menu search input.
        tabIndex={-1}
        isOptionSelected={() => false}
        className={baseClass}
        classNamePrefix={baseClass}
        searchQuery={searchQuery}
        onChangeSearchQuery={onChangeSearchQuery}
        forwardNavKey={forwardNavKey}
        onClickAddTeam={onClickAddTeam}
        showAddTeamButton={!!isGlobalAdmin}
        showSearch={showSearch}
        noOptionsMessage={() => "No matching fleets"}
      />
    </div>
  );
};

export default TeamsDropdown;
