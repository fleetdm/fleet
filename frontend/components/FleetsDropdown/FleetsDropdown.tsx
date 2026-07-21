import React, {
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import Select, {
  components,
  GroupBase,
  MenuListProps,
  MenuProps,
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
  // Generic parameter *names* must match react-select's own Props interface
  // AND every other augmentation of it in the codebase (TS2428) — do not
  // rename or underscore-prefix. IsMulti + Group are unused here by name;
  // silenced with eslint-disable comments instead.
  export interface Props<
    Option,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    IsMulti extends boolean,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    Group extends GroupBase<Option>
  > {
    searchQuery?: string;
    onChangeSearchQuery?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    // Forwards navigation keys (Arrow/Enter/Escape) from the in-menu search
    // input to react-select's own (hidden) input so option highlighting and
    // selection still work while the search input has focus.
    forwardNavKey?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
    onClickAddFleet?: () => void;
    showAddFleetButton?: boolean;
    showSearch?: boolean;
  }
}

// Search input only appears once the list is long enough to be worth
// filtering. The "Add fleet" footer always renders for global admins,
// regardless of how many fleets exist.
const MIN_FLEETS_FOR_SEARCH = 10;

export interface INumberDropdownOption extends Omit<IDropdownOption, "value"> {
  value: number;
}

const generateDropdownOptions = (
  teams: ITeamSummary[] | undefined,
  includeAllFleets: boolean,
  includeUnassigned?: boolean
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
        (o.label === APP_CONTEXT_NO_TEAM_SUMMARY.name && !includeUnassigned) ||
        (o.label === APP_CONTEXT_ALL_TEAMS_SUMMARY.name && !includeAllFleets)
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

interface IFleetsDropdownProps {
  currentUserTeams: ITeamSummary[];
  selectedFleetId?: number;
  includeAllFleets?: boolean;
  includeUnassigned?: boolean;
  isDisabled?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
  /**
   * Optional callback fired when a global admin clicks "+ Add fleet". Lets
   * the consuming page navigate with its own `router.push` (recommended)
   * instead of the fallback `browserHistory.push(PATHS.ADMIN_FLEETS + …)`.
   */
  onAddFleet?: () => void;
  /** Indicates that this fleets dropdown should be styled as a form field */
  asFormField?: boolean;
}

const baseClass = "fleet-dropdown";

// Custom Menu wraps the search input (above) and the "Add fleet" footer
// (below) *outside* the scrolling MenuList. Keeping them out of the scroll
// container means the native scrollbar spans only the options area — it
// doesn't run behind the sticky search or the sticky footer.
const CustomMenu = (props: MenuProps<INumberDropdownOption, false>) => {
  const { selectProps } = props;
  const {
    searchQuery,
    onChangeSearchQuery,
    forwardNavKey,
    onClickAddFleet,
    showAddFleetButton,
    showSearch,
  } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Autofocus via native browser behavior. react-select unmounts / remounts
  // <components.Menu> across every open/close cycle, so the input is a fresh
  // DOM node on each open — autoFocus fires reliably each time without
  // relying on effect deps semantics.

  const handleInputClick = (
    event: React.MouseEvent<HTMLInputElement, MouseEvent>
  ) => {
    inputRef.current?.focus();
    event.stopPropagation();
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (NAV_KEYS.has(event.key)) {
      event.preventDefault();
      forwardNavKey?.(event);
      return;
    }
    event.stopPropagation();
  };

  const addFleetMouseDown = (event: React.MouseEvent) => {
    // Keep focus out of react-select's hidden input so the click fires on a
    // still-mounted menu.
    event.preventDefault();
    event.stopPropagation();
  };

  return (
    <components.Menu {...props}>
      {showSearch && (
        <div className={`${baseClass}__search-row`}>
          <div className={`${baseClass}__search-field`}>
            <input
              ref={inputRef}
              // eslint-disable-next-line jsx-a11y/no-autofocus
              autoFocus
              className={`${baseClass}__search-input`}
              value={searchQuery ?? ""}
              name="fleet-search-input"
              type="text"
              placeholder="Search fleets"
              onKeyDown={handleKeyDown}
              onChange={onChangeSearchQuery}
              onClick={handleInputClick}
              onMouseDown={(event) => event.stopPropagation()}
            />
            <Icon name="search" />
          </div>
        </div>
      )}
      {props.children}
      {showAddFleetButton && (
        <div
          className={`${baseClass}__add-fleet-footer`}
          onMouseDown={addFleetMouseDown}
        >
          <Button
            variant="brand-inverse-icon"
            onClick={onClickAddFleet}
            iconStroke
            size="small"
          >
            <>
              Add fleet
              <Icon name="plus" color="core-fleet-green" />
            </>
          </Button>
        </div>
      )}
    </components.Menu>
  );
};

// CustomMenuList only wraps the option list + a sticky scroll-fade at the
// bottom. Because search + footer moved to CustomMenu, this element is the
// full scroll container and its scrollbar spans only the options.
const CustomMenuList = (props: MenuListProps<INumberDropdownOption, false>) => {
  const menuListElRef = useRef<HTMLDivElement | null>(null);
  const [hasMoreBelow, setHasMoreBelow] = useState(false);

  const updateHasMoreBelow = () => {
    const el = menuListElRef.current;
    if (!el) return;
    setHasMoreBelow(el.scrollHeight - el.scrollTop - el.clientHeight > 1);
  };

  const setMenuListRef = (el: HTMLDivElement | null) => {
    menuListElRef.current = el;
    // Chain react-select's own innerRef so its scroll-to-highlighted-option
    // logic keeps working.
    props.innerRef?.(el as HTMLDivElement);
  };

  // Measure whether the options list is scrollable after layout — the
  // ref-callback path fires before layout, so scrollHeight / clientHeight
  // can both read 0 on the first render and the fade wouldn't appear at
  // all. Keying on the child count avoids re-measuring on unrelated
  // renders (e.g. every keystroke inside the search input); the onScroll
  // handler covers user-driven position changes.
  useLayoutEffect(() => {
    updateHasMoreBelow();
  }, [React.Children.count(props.children)]);

  return (
    <components.MenuList
      {...props}
      innerRef={setMenuListRef}
      innerProps={{
        ...props.innerProps,
        onScroll: updateHasMoreBelow,
        onMouseDown: (event: React.MouseEvent) => event.stopPropagation(),
        // Chrome (and other browsers with `keyboard-focusable-scrollers`
        // enabled) auto-focuses scrollable containers to allow keyboard
        // scrolling — that steals Tab from the search input and lands on
        // an outlined MenuList instead of the "Add fleet" button. tabIndex
        // -1 opts out; the search input + forwardNavKey bridge already
        // handle keyboard nav through options.
        tabIndex: -1,
      }}
    >
      {props.children}
      {hasMoreBelow && (
        <div className={`${baseClass}__scroll-fade`} aria-hidden />
      )}
    </components.MenuList>
  );
};

const FleetsDropdown = ({
  currentUserTeams,
  selectedFleetId,
  includeAllFleets = true,
  includeUnassigned = false,
  isDisabled = false,
  onChange,
  onOpen,
  onClose,
  onAddFleet,
  asFormField = false,
}: IFleetsDropdownProps): JSX.Element => {
  const { isGlobalAdmin, config } = useContext(AppContext);

  // Mirror the gate used by ManageFleetsPage's in-page "Add fleet" button:
  // Primo mode and GitOps mode both disable fleet creation. Hide the
  // affordance entirely rather than link to a page where the button is
  // disabled — otherwise the deep link (?create_fleet=1) silently no-ops
  // when it lands.
  const isAddFleetDisabled =
    !!config?.partnerships?.enable_primo ||
    !!(config?.gitops?.gitops_mode_enabled && config?.gitops?.repository_url);
  const canAddFleet = !!isGlobalAdmin && !isAddFleetDisabled;
  const [searchQuery, setSearchQuery] = useState("");
  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const selectRef = useRef<SelectInstance<INumberDropdownOption, false>>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const fleetOptions: INumberDropdownOption[] = useMemo(
    () =>
      generateDropdownOptions(
        currentUserTeams,
        includeAllFleets,
        includeUnassigned
      ),
    [currentUserTeams, includeAllFleets, includeUnassigned]
  );

  const filteredOptions = useMemo(
    () => filterOptionsBySearch(fleetOptions, searchQuery),
    [fleetOptions, searchQuery]
  );

  const showSearch = fleetOptions.length >= MIN_FLEETS_FOR_SEARCH;

  const selectedValue = fleetOptions.find(
    (option) => selectedFleetId === option.value
  )
    ? selectedFleetId
    : fleetOptions[0]?.value;

  const selectedLabel =
    fleetOptions.find((o) => o.value === selectedValue)?.label ?? "All fleets";

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

  // When the menu opens with no search input, focus react-select's hidden
  // input directly so Arrow / Enter / Escape drive option highlighting
  // natively — otherwise focus stays on the trigger and keydowns never reach
  // react-select. When search is visible, CustomMenuList focuses the search
  // input on mount and the forwardNavKey bridge takes it from there.
  useEffect(() => {
    if (!menuIsOpen || showSearch) return;
    const hiddenInput = ((selectRef.current as unknown) as {
      inputRef?: HTMLInputElement | null;
    })?.inputRef;
    hiddenInput?.focus();
  }, [menuIsOpen, showSearch]);

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

  const onClickAddFleet = () => {
    setMenuIsOpen(false);
    setSearchQuery("");
    onClose?.();
    if (onAddFleet) {
      onAddFleet();
    } else {
      // Fallback for callers that haven't wired their own router-based
      // navigation. `browserHistory` is inconsistent with the router prop
      // pattern used elsewhere; consumers should migrate to onAddFleet.
      browserHistory.push(`${PATHS.ADMIN_FLEETS}?create_fleet=1`);
    }
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
      borderRadius: "8px",
      // Page-overlay tier (99) per the 9/99/999 z-index convention.
      zIndex: 99,
      overflow: "hidden",
      border: 0,
      marginTop: PADDING["pad-xsmall"],
      width: 340,
      // Cap total menu height so the whole dropdown (search + options list +
      // footer) fits 14 options before the scrollbar engages — scroll first
      // shows at 15 fleets per design. `min(...)` also clamps against the
      // viewport with ~32px breathing room — the design's "or when
      // restricted by page height" clause.
      maxHeight: "min(715px, calc(100vh - 32px))",
      // Menu owns the outer pad-medium inset; the search-row provides the
      // pad-medium gap below the input, and the footer's padding-top
      // provides the pad-medium above the "Add fleet" button. The options
      // list abuts the footer's border-top directly (no gap in between).
      padding: PADDING["pad-medium"],
      display: "flex",
      flexDirection: "column",
      position: "absolute",
      left: 0,
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (baseStyles) => ({
      ...baseStyles,
      // The scrolling area. Fills remaining height inside the Menu flex
      // column so the scrollbar spans only the options — never behind the
      // (Menu-level) search-row or "Add fleet" footer.
      flex: "1 1 auto",
      minHeight: 0,
      overflowY: "auto",
      maxHeight: "none",
      // Menu owns the outer horizontal padding; a pad-small paddingBottom
      // gives the last option a bit of breathing room above the footer's
      // divider when the list is scrolled to the end.
      padding: `0 0 ${PADDING["pad-small"]}`,
      position: "relative",
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
        value={fleetOptions.find((option) => option.value === selectedValue)}
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
          Menu: CustomMenu,
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
        onClickAddFleet={onClickAddFleet}
        showAddFleetButton={canAddFleet}
        showSearch={showSearch}
        noOptionsMessage={() => "No matching fleets"}
      />
    </div>
  );
};

export default FleetsDropdown;
