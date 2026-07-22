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
import { getPathWithQueryParams } from "utilities/url";
import { IDropdownOption } from "interfaces/dropdownOption";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  APP_CONTEXT_NO_TEAM_ID,
  ITeamSummary,
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

// Search input only appears once the option list has this many rows or more
// — rows include "All fleets" and "Unassigned" alongside real fleets, so the
// threshold is measured in rows rather than fleets. The "Add fleet" footer
// always renders for global admins, regardless of row count.
const MIN_ROWS_FOR_SEARCH = 10;

export interface INumberDropdownOption extends Omit<IDropdownOption, "value"> {
  value: number;
}

const generateDropdownOptions = (
  fleets: ITeamSummary[] | undefined,
  includeAllFleets: boolean,
  includeUnassigned?: boolean
): INumberDropdownOption[] => {
  if (!fleets) return [];

  const options: INumberDropdownOption[] = fleets.map((fleet) => ({
    disabled: false,
    label: fleet.name,
    value: fleet.id,
  }));

  // Filter the synthetic rows by ID (stable), not label — a real fleet
  // could legitimately be named "All fleets" or "Unassigned" and would
  // otherwise get dropped by a label-based check.
  return options.filter(
    (o) =>
      !(
        (o.value === APP_CONTEXT_NO_TEAM_ID && !includeUnassigned) ||
        (o.value === APP_CONTEXT_ALL_TEAMS_ID && !includeAllFleets)
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
  currentUserFleets: ITeamSummary[];
  selectedFleetId?: number;
  includeAllFleets?: boolean;
  includeUnassigned?: boolean;
  isDisabled?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
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

  const handleInputClick = (
    event: React.MouseEvent<HTMLInputElement, MouseEvent>
  ) => {
    inputRef.current?.focus();
    event.stopPropagation();
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    // Stop propagation so the original event doesn't ALSO bubble to
    // SelectContainer and get processed a second time (double-fire on
    // Enter/Arrow). Nav keys are forwarded explicitly via forwardNavKey.
    event.stopPropagation();
    if (NAV_KEYS.has(event.key)) {
      event.preventDefault();
      forwardNavKey?.(event);
    }
  };

  const addFleetMouseDown = (event: React.MouseEvent) => {
    // Keep focus out of react-select's hidden input so the click fires on a
    // still-mounted menu.
    event.preventDefault();
    event.stopPropagation();
  };

  const addFleetKeyDown = (event: React.KeyboardEvent) => {
    // Stop Enter/Space from bubbling to SelectContainer, which would treat
    // them as "select highlighted option" alongside the button's own click.
    // Escape/Tab/Arrow still bubble so react-select's close/focus work.
    // preventDefault on Enter — Fleet Button's handleKeyDown already
    // synthesizes onClick from Enter, so without preventDefault the browser
    // would ALSO synthesize a native click and fire onClickAddFleet twice.
    if (event.key === "Enter") {
      event.preventDefault();
      event.stopPropagation();
    } else if (event.key === " ") {
      event.stopPropagation();
    }
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
              type="text"
              placeholder="Search fleets"
              aria-label="Search fleets"
              autoComplete="off"
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
          onKeyDown={addFleetKeyDown}
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
  const childCount = React.Children.count(props.children);
  useLayoutEffect(() => {
    updateHasMoreBelow();
  }, [childCount]);

  // Chain react-select's own innerProps handlers before running our own —
  // otherwise our overrides silently drop whatever react-select (or a
  // future prop) provides.
  const originalOnScroll = props.innerProps?.onScroll;
  const originalOnMouseDown = props.innerProps?.onMouseDown;

  return (
    <components.MenuList
      {...props}
      innerRef={setMenuListRef}
      innerProps={{
        ...props.innerProps,
        onScroll: (event: React.UIEvent<HTMLDivElement>) => {
          originalOnScroll?.(event);
          updateHasMoreBelow();
        },
        onMouseDown: (event: React.MouseEvent<HTMLDivElement>) => {
          originalOnMouseDown?.(event);
          event.stopPropagation();
        },
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
      {/*
        Anchor is always rendered at 0 flow height so toggling the fade
        doesn't shift scrollHeight (which would clamp scrollTop and jump
        at the bottom). Gradient is a ::before pseudo, opacity-toggled
        via --visible.
      */}
      <div
        className={classnames(`${baseClass}__scroll-fade`, {
          [`${baseClass}__scroll-fade--visible`]: hasMoreBelow,
        })}
        aria-hidden
      />
    </components.MenuList>
  );
};

const FleetsDropdown = ({
  currentUserFleets,
  selectedFleetId,
  includeAllFleets = true,
  includeUnassigned = false,
  isDisabled = false,
  onChange,
  onOpen,
  onClose,
  asFormField = false,
}: IFleetsDropdownProps): JSX.Element => {
  const { isGlobalAdmin, config } = useContext(AppContext);

  // Mirrors ManageFleetsPage: Primo + GitOps disable fleet creation. Also
  // hide when asFormField — clicking Add fleet would abandon in-progress
  // form input.
  const isPrimoModeEnabled = !!config?.partnerships?.enable_primo;
  const isGitOpsModeEnabled = !!(
    config?.gitops?.gitops_mode_enabled && config?.gitops?.repository_url
  );
  const isAddFleetDisabled = isPrimoModeEnabled || isGitOpsModeEnabled;
  const canAddFleet = !!isGlobalAdmin && !isAddFleetDisabled && !asFormField;
  const [searchQuery, setSearchQuery] = useState("");
  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const selectRef = useRef<SelectInstance<INumberDropdownOption, false>>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // react-select's SelectInstance doesn't type `inputRef` publicly.
  // Centralize the cast — if react-select ever renames this field, both
  // call sites (focus effect + forwardNavKey bridge) fail together. The
  // dev-only warning surfaces the loss of keyboard nav loudly on a
  // react-select upgrade instead of silently regressing.
  const getHiddenInput = () => {
    const ref = selectRef.current;
    if (!ref) return null;
    const input = ((ref as unknown) as {
      inputRef?: HTMLInputElement | null;
    }).inputRef;
    if (process.env.NODE_ENV !== "production" && input === undefined) {
      // eslint-disable-next-line no-console
      console.warn(
        "FleetsDropdown: react-select's SelectInstance is missing the expected `inputRef` field. Keyboard nav may not work."
      );
    }
    return input ?? null;
  };

  const fleetOptions: INumberDropdownOption[] = useMemo(
    () =>
      generateDropdownOptions(
        currentUserFleets,
        includeAllFleets,
        includeUnassigned
      ),
    [currentUserFleets, includeAllFleets, includeUnassigned]
  );

  const filteredOptions = useMemo(
    () => filterOptionsBySearch(fleetOptions, searchQuery),
    [fleetOptions, searchQuery]
  );

  const showSearch = fleetOptions.length >= MIN_ROWS_FOR_SEARCH;

  const selectedValue = fleetOptions.find(
    (option) => selectedFleetId === option.value
  )
    ? selectedFleetId
    : fleetOptions[0]?.value;

  const selectedLabel =
    fleetOptions.find((o) => o.value === selectedValue)?.label ??
    APP_CONTEXT_ALL_TEAMS_SUMMARY.name;

  // Close menu on click outside. Only attach the listener while the menu
  // is open. The transition effect below owns searchQuery clearing.
  useEffect(() => {
    if (!menuIsOpen) return undefined;
    const handleClickOutside = (event: MouseEvent) => {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(event.target as Node)
      ) {
        setMenuIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [menuIsOpen]);

  // When the menu opens with no search input, focus react-select's hidden
  // input directly so Arrow / Enter / Escape drive option highlighting
  // natively — otherwise focus stays on the trigger and keydowns never
  // reach react-select. When search IS visible, the search input's native
  // `autoFocus` (in CustomMenu) handles focus, and the forwardNavKey
  // bridge routes nav keys through to react-select's hidden input.
  useEffect(() => {
    if (!menuIsOpen || showSearch) return;
    getHiddenInput()?.focus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [menuIsOpen, showSearch]);

  // Fire onClose + clear searchQuery once per true -> false transition.
  // react-select's own onMenuClose only fires on its own closes; this
  // effect catches all paths (controlled and library-driven), so we can
  // stop repeating the same setSearchQuery("") + onClose at each close
  // origin. onClose is stashed in a ref so an inline parent callback
  // doesn't retrigger this effect.
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);
  const wasOpenRef = useRef(false);
  useEffect(() => {
    if (menuIsOpen) {
      wasOpenRef.current = true;
    } else if (wasOpenRef.current) {
      wasOpenRef.current = false;
      setSearchQuery("");
      onCloseRef.current?.();
    }
  }, [menuIsOpen]);

  const toggleMenu = () => {
    if (isDisabled) return;
    // Keep side effects out of the state updater — Strict Mode runs
    // updaters twice, which would double-fire onOpen. searchQuery clear
    // + onClose fire from the transition effect above.
    if (menuIsOpen) {
      setMenuIsOpen(false);
    } else {
      setMenuIsOpen(true);
      onOpen?.();
    }
  };

  const handleChange = (newValue: INumberDropdownOption | null) => {
    if (!newValue) return;
    onChange(newValue.value);
    setMenuIsOpen(false);
  };

  const onChangeSearchQuery = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(event.target.value);
  };

  // Forwards a navigation key from the in-menu search input to react-select's
  // hidden input so its built-in keyDown handler runs (option highlighting,
  // selection, menu close).
  const forwardNavKey = (event: React.KeyboardEvent<HTMLInputElement>) => {
    const input = getHiddenInput();
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
    // TODO: hoist navigation to an onAddFleet callback prop so consumers own it.
    browserHistory.push(
      getPathWithQueryParams(PATHS.ADMIN_FLEETS, { create_fleet: "1" })
    );
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
      // shows at 15 rows per design. `min(...)` also clamps against the
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
        onMenuClose={() => setMenuIsOpen(false)}
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
