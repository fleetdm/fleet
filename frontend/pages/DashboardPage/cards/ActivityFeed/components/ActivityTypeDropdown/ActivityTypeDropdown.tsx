import React, { useRef, useMemo } from "react";
import classnames from "classnames";
import Select, {
  components,
  MenuListProps,
  SelectInstance,
  SingleValue,
  StylesConfig,
  GroupBase,
  ValueContainerProps,
} from "react-select-5";

import {
  ACTIVITY_TYPE_TO_FILTER_LABEL,
  ActivityType,
} from "interfaces/activity";

import {
  CustomOptionType,
  CustomDropdownIndicator,
  generateCustomDropdownStyles,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Icon from "components/Icon";
import FormField from "components/forms/FormField";

import { PADDING } from "styles/var/padding";
import { FONT_SIZES } from "styles/var/fonts";
import { COLORS } from "styles/var/colors";

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

const baseClass = "activity-type-dropdown";

type ICustomMenuListProps = MenuListProps<CustomOptionType, false>;

const CustomMenuList = (props: ICustomMenuListProps) => {
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
          name="label-search-input"
          type="text"
          placeholder="e.g. wiped host"
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
      {props.children}
    </components.MenuList>
  );
};

const CustomValueContainer = ({
  children,
  ...props
}: ValueContainerProps<CustomOptionType, false>) => {
  return (
    <components.ValueContainer {...props}>
      <Icon name="filter-alt" className="filter-icon" />
      {children}
    </components.ValueContainer>
  );
};

const TYPE_FILTER_OPTIONS: CustomOptionType[] = Object.values(ActivityType)
  .map((type) => ({
    label: ACTIVITY_TYPE_TO_FILTER_LABEL[type],
    value: type,
  }))
  .sort((a, b) => a.label.localeCompare(b.label));

TYPE_FILTER_OPTIONS.unshift({
  label: "All types",
  value: "all",
});

const generateOptions = (searchQuery: string) => {
  const query = searchQuery.toLowerCase().trim();
  if (query === "") {
    return TYPE_FILTER_OPTIONS;
  }

  return TYPE_FILTER_OPTIONS.filter((option) => {
    if (typeof option.label !== "string") {
      return false;
    }
    return option.label.toLowerCase().includes(query);
  });
};

interface IActivityTypeDropdownProps {
  value: string;
  onSelect: (value: string) => void;
  className?: string;
}

const ActivityTypeDropdown = ({
  value,
  onSelect,
  className,
}: IActivityTypeDropdownProps) => {
  const [searchQuery, setSearchQuery] = React.useState("");
  const [menuIsOpen, setMenuIsOpen] = React.useState(false);

  const selectRef = useRef<SelectInstance<CustomOptionType, false>>(null);
  const isSearchInputFocusedRef = useRef(false);

  const handleChange = (option: SingleValue<CustomOptionType>) => {
    if (option === null) return;

    setSearchQuery("");
    onSelect(option.value !== "all" ? option.value : "");
    selectRef.current?.blur();
  };

  const onChangeSearchQuery = (event: React.ChangeEvent<HTMLInputElement>) => {
    // We need to stop the key presses propagation to prevent the dropdown from
    // picking up keypresses.
    event.stopPropagation();
    setSearchQuery(event.target.value);
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setMenuIsOpen(false);
      selectRef.current?.blur();
    } else if (e.key === "Tab" && !e.shiftKey) {
      // Allow tabbing out of the component
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
    }
  };

  const onClickSearchInput = () => {
    isSearchInputFocusedRef.current = true;
  };

  const onBlurSearchInput = () => {
    isSearchInputFocusedRef.current = false;
  };

  const toggleMenu = () => {
    menuIsOpen && selectRef.current?.blur();
    setMenuIsOpen(!menuIsOpen);
  };

  const getValue = () => {
    return (
      TYPE_FILTER_OPTIONS.find((option) => option.value === value) ||
      TYPE_FILTER_OPTIONS[0]
    );
  };

  const customStyles: StylesConfig<CustomOptionType, false> = {
    // TODO: update the generateCustomDropdownStyles to override styles for the components.
    // Right now, we have to copy over the entire styles object for that component to make small adjustments.
    ...generateCustomDropdownStyles(),
    menu: (provided) => ({
      ...provided,
      boxShadow: "0 2px 6px rgba(0, 0, 0, 0.1)",
      borderRadius: "4px",
      zIndex: 6,
      overflow: "hidden",
      border: 0,
      marginTop: "3px",
      left: 0,
      maxHeight: "none",
      position: "absolute",
      animation: "fade-in 150ms ease-out",
      width: 370,
    }),
    menuList: (provided) => ({
      ...provided,
      maxHeight: 360,
      // we want to remove the padding from the top and handle that with the search field.
      // This ensures the scrolled options dont show above a gap above the search field.
      paddingBottom: PADDING["pad-small"],
      paddingLeft: PADDING["pad-small"],
      paddingRight: PADDING["pad-small"],
      paddingTop: 0,
    }),
    noOptionsMessage: (provided) => ({
      ...provided,
      padding: PADDING["pad-small"],
      fontSize: FONT_SIZES["x-small"],
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: "100%",
      height: 276,
      color: COLORS["ui-fleet-black-75"],
    }),
    valueContainer: (provided) => ({
      ...provided,
      padding: 0,
      display: "flex",
      gap: PADDING["pad-small"],
      // we need the no wrap to keep the value and icon on the same line. The value
      // will be truncated with ellipsis if too long.
      flexWrap: "nowrap",
    }),
  };

  const classNames = classnames(baseClass, className);
  const options = useMemo(() => generateOptions(searchQuery), [searchQuery]);

  return (
    <FormField
      className={classNames}
      type="dropdown"
      name="activity-type-dropdown"
      label=""
    >
      <div onClick={toggleMenu}>
        <Select<CustomOptionType, false>
          ref={selectRef}
          classNamePrefix="activity-type-select"
          styles={customStyles}
          menuIsOpen={menuIsOpen}
          options={options}
          components={{
            MenuList: CustomMenuList,
            DropdownIndicator: CustomDropdownIndicator,
            IndicatorSeparator: () => null,
            ValueContainer: CustomValueContainer,
          }}
          isSearchable={false}
          value={getValue()}
          onChange={handleChange}
          searchQuery={searchQuery}
          noOptionsMessage={() => "No items match this search criteria."}
          onKeyDown={onKeyDown}
          onBlur={onBlur}
          onChangeSearchQuery={onChangeSearchQuery}
          onClickSearchInput={onClickSearchInput}
          onBlurSearchInput={onBlurSearchInput}
        />
      </div>
    </FormField>
  );
};

export default ActivityTypeDropdown;
