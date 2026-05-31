import React, { useMemo, useRef, useState } from "react";
import Select, {
  components,
  DropdownIndicatorProps,
  GroupBase,
  MenuListProps,
  SelectInstance,
  StylesConfig,
} from "react-select-5";

import { COLORS } from "styles/var/colors";
import { PADDING } from "styles/var/padding";
import { FONT_SIZES } from "styles/var/fonts";
import { ISelfServiceCategory } from "interfaces/self_service_category";

import Icon from "components/Icon";

declare module "react-select-5/dist/declarations/src/Select" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>
  > {
    searchQuery?: string;
    onChangeSearchQuery?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    onClickSearchInput?: React.MouseEventHandler<HTMLInputElement>;
    onBlurSearchInput?: React.FocusEventHandler<HTMLInputElement>;
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
  const {
    searchQuery,
    onChangeSearchQuery,
    onClickSearchInput,
    onBlurSearchInput,
  } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  const handleInputClick = (
    event: React.MouseEvent<HTMLInputElement, MouseEvent>
  ) => {
    onClickSearchInput && onClickSearchInput(event);
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
          onBlur={onBlurSearchInput}
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
  const isSearchInputFocusedRef = useRef(false);

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

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setMenuIsOpen(false);
      selectRef.current?.blur();
    } else if (e.key === "Tab" && !e.shiftKey) {
      setMenuIsOpen(false);
      selectRef.current?.blur();
    } else {
      setMenuIsOpen(true);
    }
  };

  const onBlur = () => {
    if (!isSearchInputFocusedRef.current) {
      isSearchInputFocusedRef.current = false;
      setMenuIsOpen(false);
      setSearchQuery("");
    }
  };

  const onClickSearchInput = () => {
    isSearchInputFocusedRef.current = true;
  };

  const onBlurSearchInput = () => {
    isSearchInputFocusedRef.current = false;
  };

  const toggleMenu = () => {
    if (menuIsOpen) {
      selectRef.current?.blur();
    }
    setMenuIsOpen(!menuIsOpen);
  };

  const CustomDropdownIndicator = (
    props: DropdownIndicatorProps<
      ICategoryOption,
      false,
      GroupBase<ICategoryOption>
    >
  ) => {
    const { isFocused, selectProps } = props;
    const color =
      isFocused || selectProps.menuIsOpen
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

  // see https://react-select.com/styles#the-styles-prop
  const customStyles: StylesConfig<ICategoryOption, false> = {
    control: (baseStyles, state) => ({
      ...baseStyles,
      padding: "4px 0",
      backgroundColor: "initial",
      border: 0,
      display: "flex",
      flexDirection: "row",
      borderRadius: "4px",
      boxShadow: "none",
      cursor: "pointer",
      minHeight: 0,
      "&:hover": {
        boxShadow: "none",
        [`.${baseClass}__single-value`]: {
          color: COLORS["core-fleet-black"],
        },
        [`.${baseClass}__indicator path`]: {
          stroke: COLORS["ui-fleet-black-75-over"],
        },
      },
      ...(state.isDisabled && {
        [`.${baseClass}__single-value`]: {
          color: COLORS["ui-fleet-black-50"],
        },
        [`.${baseClass}__indicator path`]: {
          stroke: COLORS["ui-fleet-black-50"],
        },
      }),
      ...(state.menuIsOpen && {
        [`.${baseClass}__indicator svg`]: {
          transform: "rotate(180deg)",
          transition: "transform 0.25s ease",
        },
      }),
    }),
    singleValue: (baseStyles) => ({
      ...baseStyles,
      color: COLORS["core-fleet-black"],
      lineHeight: "24px",
      paddingLeft: 0,
      paddingRight: "8px",
      margin: 0,
      fontWeight: 600,
      fontSize: "16px",
      // omit grid-column-end for automatic width — lets dropdown grow with content
      gridArea: "1/1/2",
    }),
    dropdownIndicator: (baseStyles) => ({
      ...baseStyles,
      display: "flex",
      padding: "2px",
      margin: "0 0 0 4px",
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
    valueContainer: (baseStyles) => ({
      ...baseStyles,
      padding: 0,
    }),
    input: (baseStyles) => ({
      ...baseStyles,
      margin: 0,
      color: COLORS["core-fleet-black"],
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
    <div className={`${baseClass}-wrapper`}>
      <div onClick={toggleMenu}>
        <Select<ICategoryOption, false>
          ref={selectRef}
          options={options}
          value={allOptions.find((o) => o.value === selectedValue)}
          onChange={(newValue) => {
            if (!newValue) return;
            setSearchQuery("");
            onChange(
              newValue.value === ALL_CATEGORIES_VALUE
                ? undefined
                : newValue.value
            );
            selectRef.current?.blur();
          }}
          isDisabled={isDisabled}
          isSearchable={false}
          menuIsOpen={menuIsOpen}
          styles={customStyles}
          components={{
            MenuList: CustomMenuList,
            DropdownIndicator: CustomDropdownIndicator,
            IndicatorSeparator: () => null,
          }}
          className={baseClass}
          classNamePrefix={baseClass}
          searchQuery={searchQuery}
          onChangeSearchQuery={onChangeSearchQuery}
          onClickSearchInput={onClickSearchInput}
          onBlurSearchInput={onBlurSearchInput}
          onKeyDown={onKeyDown}
          onBlur={onBlur}
          noOptionsMessage={() => "No categories match this search."}
        />
      </div>
    </div>
  );
};

export default CategoryFilter;
