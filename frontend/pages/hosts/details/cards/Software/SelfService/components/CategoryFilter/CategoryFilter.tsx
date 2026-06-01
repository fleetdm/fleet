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
import { FONT_SIZES } from "styles/var/fonts";
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

const CustomMenuList = (props: MenuListProps<ICategoryOption, false>) => {
  const { selectProps } = props;
  const { searchQuery, onChangeSearchQuery } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  const handleInputClick = (
    event: React.MouseEvent<HTMLInputElement, MouseEvent>
  ) => {
    inputRef.current?.focus();
    event.stopPropagation();
  };

  return (
    <components.MenuList {...props}>
      <div className={`${baseClass}__search-field`}>
        <input
          className={`${baseClass}__search-input`}
          ref={inputRef}
          value={searchQuery}
          name="category-search-input"
          type="text"
          placeholder="Search categories"
          onKeyDown={(event) => {
            // Stops the parent dropdown from picking up on input keypresses
            event.stopPropagation();
          }}
          onChange={onChangeSearchQuery}
          onClick={handleInputClick}
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
  // ActionsDropdown brand-button pattern in components/ActionsDropdown).
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
    // Keep "All" pinned at the top; filter the rest by label
    return allOptions.filter(
      (option) =>
        option.value === ALL_CATEGORIES_VALUE ||
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
  };

  const selectedLabel =
    allOptions.find((o) => o.value === selectedValue)?.label ?? "All";

  // see https://react-select.com/styles#the-styles-prop
  // Control + DropdownIndicator are replaced by a real Fleet <Button> above,
  // and SingleValue is rendered as plain text in that button — so we only
  // need to style the menu/option subtree here.
  const customStyles: StylesConfig<ICategoryOption, false> = {
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
      minWidth: "240px",
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
      padding: PADDING["pad-small"],
      fontSize: FONT_SIZES["x-small"],
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
        components.Control: () => null. This mirrors the brand-button
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
          Control: () => null,
          DropdownIndicator: () => null,
          IndicatorSeparator: () => null,
        }}
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
