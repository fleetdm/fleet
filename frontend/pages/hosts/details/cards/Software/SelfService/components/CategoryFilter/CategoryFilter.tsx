import React, { useEffect, useMemo, useRef, useState } from "react";
import Select, {
  components,
  GroupBase,
  MenuListProps,
  SelectInstance,
  StylesConfig,
} from "react-select-5";

import { COLORS } from "styles/var/colors";
import { PADDING } from "styles/var/padding";
import { ISelfServiceCategory } from "interfaces/self_service_category";

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
  }
}

const baseClass = "self-service-category-filter";

export const ALL_CATEGORIES_VALUE = -1;

interface ICategoryOption {
  label: string;
  value: number;
}

export interface ICategoryFilterProps {
  categories: ISelfServiceCategory[];
  selectedCategoryId?: number;
  onChange: (newCategoryId: number | undefined) => void;
  isDisabled?: boolean;
}

const NAV_KEYS = new Set(["ArrowDown", "ArrowUp", "Enter", "Escape"]);

const CustomMenuList = (props: MenuListProps<ICategoryOption, false>) => {
  const { selectProps } = props;
  const { searchQuery, onChangeSearchQuery, forwardNavKey } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Auto-focus the search input when the menu opens so keyboard users can
  // start typing immediately.
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

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
    // Block other keys from bubbling to react-select (which would otherwise
    // try to type-ahead-match options).
    event.stopPropagation();
  };

  // Block clicks on the MenuList's own padding from bubbling to the Menu
  // element's onMenuMouseDown handler in react-select (lib line 1428), which
  // would otherwise call focusInput() and steal focus to the hidden Control
  // input — making a subsequent click on the search input look like a close.
  // Option clicks still fire because Option's own handler runs at the option
  // level before bubbling reaches MenuList.
  return (
    <components.MenuList
      {...props}
      innerProps={{
        ...props.innerProps,
        onMouseDown: (event: React.MouseEvent) => event.stopPropagation(),
      }}
    >
      <div className={`${baseClass}__search-field`}>
        <input
          className={`${baseClass}__search-input`}
          ref={inputRef}
          value={searchQuery}
          name="category-search-input"
          type="text"
          placeholder="Search categories"
          onKeyDown={handleKeyDown}
          onChange={onChangeSearchQuery}
          onClick={handleInputClick}
          // react-select attaches onMouseDown to the Menu element that calls
          // preventDefault + focusInput on its own hidden input — stealing
          // focus from anything you click inside the menu. Stopping the
          // synthetic event here keeps focus on our search input.
          onMouseDown={(event) => event.stopPropagation()}
        />
        <Icon name="search" />
      </div>
      <div className={`${baseClass}__options-spacer`} />
      {props.children}
    </components.MenuList>
  );
};

const CategoryFilter = ({
  categories,
  selectedCategoryId,
  onChange,
  isDisabled,
}: ICategoryFilterProps) => {
  const [searchQuery, setSearchQuery] = useState("");
  const [menuIsOpen, setMenuIsOpen] = useState(false);

  const selectRef = useRef<SelectInstance<ICategoryOption, false>>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // Close menu when clicking outside the wrapper (mirrors the
  // ActionsDropdown primary-variant pattern in components/ActionsDropdown).
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
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [menuIsOpen]);

  const allOptions: ICategoryOption[] = useMemo(
    () => [
      { label: "All", value: ALL_CATEGORIES_VALUE },
      ...categories.map((category) => ({
        label: category.name,
        value: category.id,
      })),
    ],
    [categories]
  );

  const options = useMemo(() => {
    const query = searchQuery.toLowerCase().trim();
    if (query === "") return allOptions;
    return allOptions.filter((option) =>
      option.label.toLowerCase().includes(query)
    );
  }, [allOptions, searchQuery]);

  const selectedValue =
    selectedCategoryId !== undefined
      ? selectedCategoryId
      : ALL_CATEGORIES_VALUE;

  const onChangeSearchQuery = (event: React.ChangeEvent<HTMLInputElement>) => {
    event.stopPropagation();
    setSearchQuery(event.target.value);
  };

  const toggleMenu = () => {
    setMenuIsOpen((open) => !open);
    // Focus moves to the in-menu search input via CustomMenuList's mount
    // effect; arrow/enter/escape get forwarded from there to react-select.
  };

  // Forwards a navigation key from the in-menu search input to react-select's
  // hidden input so its built-in keyDown handler runs (option highlighting,
  // selection, menu close). React 17+ root delegation picks up the native
  // KeyboardEvent and fires react-select's onKeyDown synthetic handler.
  const forwardNavKey = (event: React.KeyboardEvent<HTMLInputElement>) => {
    // SelectInstance exposes inputRef; cast is needed because v5's public
    // typings don't surface it.
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

  const selectedLabel =
    allOptions.find((o) => o.value === selectedValue)?.label ?? "All";

  // see https://react-select.com/styles#the-styles-prop
  // Control + DropdownIndicator are replaced by a real Fleet <Button> above,
  // and SingleValue is rendered as plain text in that button — so we only
  // need to style the menu/option subtree here.
  const customStyles: StylesConfig<ICategoryOption, false> = {
    // Hide react-select's own control (which contains the input) behind the
    // visible Button, but keep it in the DOM so its input can receive focus
    // and react-select's keyDown handler fires for ArrowUp/Down/Enter.
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
      // Clears the trigger Button's :focus-visible outline (1px ring with
      // 1px offset = 2px beyond the button) so the menu sits flush against
      // the outline rather than overlapping it.
      marginTop: PADDING["pad-xsmall"],
      width: "340px",
      maxHeight: "none",
      position: "absolute",
      left: "0",
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (baseStyles) => ({
      ...baseStyles,
      maxHeight: 360,
      // top padding is handled by the search field's own padding-top so that
      // options scrolling up are hidden by the sticky search field's background
      paddingBottom: PADDING["pad-small"],
      paddingLeft: PADDING["pad-small"],
      paddingRight: PADDING["pad-small"],
      paddingTop: 0,
    }),
    noOptionsMessage: (baseStyles) => ({
      ...baseStyles,
      // Match an option's vertical padding + font-size so the menu height
      // doesn't jump between options and the no-match message. Drop the
      // horizontal padding so the message stays on one line.
      padding: "10px 0",
      fontSize: "14px",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: "100%",
      color: COLORS["ui-fleet-black-75"],
    }),
    option: (baseStyles, state) => ({
      ...baseStyles,
      padding: "10px 8px",
      fontSize: "14px",
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
    <div className={`${baseClass}-wrapper`} ref={wrapperRef}>
      {/*
        A real Fleet <Button> renders the visible trigger so it inherits
        the Button's :focus-visible outline (Button/_styles.scss
        button-variant mixin). react-select's own Control is hidden via
        components.Control: () => null. This mirrors the primary variant
        path in components/ActionsDropdown/ActionsDropdown.tsx.
      */}
      <Button
        variant="unstyled"
        type="button"
        onClick={toggleMenu}
        disabled={isDisabled}
        className={`${baseClass}__button`}
        ariaHasPopup="listbox"
        ariaExpanded={menuIsOpen}
      >
        <span className={`${baseClass}__button-label`}>{selectedLabel}</span>
        <Icon
          name="chevron-down"
          color={menuIsOpen ? "core-fleet-black" : "ui-fleet-black-75"}
          className={`${baseClass}__icon${
            menuIsOpen ? ` ${baseClass}__icon--open` : ""
          }`}
        />
      </Button>
      <Select<ICategoryOption, false>
        ref={selectRef}
        options={options}
        value={allOptions.find((o) => o.value === selectedValue)}
        onChange={(newValue) => {
          if (!newValue) return;
          setSearchQuery("");
          onChange(
            newValue.value === ALL_CATEGORIES_VALUE ? undefined : newValue.value
          );
          setMenuIsOpen(false);
        }}
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
        forwardNavKey={forwardNavKey}
        className={baseClass}
        classNamePrefix={baseClass}
        searchQuery={searchQuery}
        onChangeSearchQuery={onChangeSearchQuery}
        noOptionsMessage={() => "No categories match this search."}
      />
    </div>
  );
};

export default CategoryFilter;
