import classnames from "classnames";
import React, { useEffect, useRef, useMemo } from "react";
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
  CustomOptionType,
  CustomDropdownIndicator,
  generateCustomDropdownStyles,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import FormField from "components/forms/FormField";
import Icon from "components/Icon";
import {
  ACTIVITY_TYPE_TO_FILTER_LABEL,
  ActivityType,
} from "interfaces/activity";
import { COLORS } from "styles/var/colors";
import { FONT_SIZES } from "styles/var/fonts";
import { PADDING } from "styles/var/padding";

declare module "react-select-5/dist/declarations/src/Select" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>
  > {
    searchQuery?: string;
    onChangeSearchQuery?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    forwardNavKey?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  }
}

const baseClass = "activity-type-dropdown";

const NAV_KEYS = new Set(["ArrowDown", "ArrowUp", "Enter", "Escape", "Tab"]);

type ICustomMenuListProps = MenuListProps<CustomOptionType, false>;

const CustomMenuList = (props: ICustomMenuListProps) => {
  const { selectProps } = props;
  const { searchQuery, onChangeSearchQuery, forwardNavKey } = selectProps;
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Not autoFocus: react-select's componentDidUpdate refocuses its own input
  // when the menu opens, which runs after autoFocus but before passive effects.
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
    // Typing must not reach react-select; nav keys are forwarded explicitly
    // so option highlighting, selection and closing still work.
    event.stopPropagation();
    if (NAV_KEYS.has(event.key)) {
      if (event.key !== "Tab") event.preventDefault();
      forwardNavKey?.(event);
    }
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
          aria-label="Search activity types"
          autoComplete="off"
          onKeyDown={handleKeyDown}
          onChange={onChangeSearchQuery}
          onClick={handleInputClick}
          // react-select's menu mousedown handler would otherwise steal focus
          // back to its hidden input.
          onMouseDown={(event) => event.stopPropagation()}
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
      <Icon name="filter" className="filter-icon" />
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
  const wrapperRef = useRef<HTMLDivElement>(null);

  // Focus moves into the search input while the menu is open, so react-select's
  // blur can't be used to detect clicks outside.
  useEffect(() => {
    if (!menuIsOpen) return undefined;
    const handleClickOutside = (event: MouseEvent) => {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(event.target as Node)
      ) {
        setMenuIsOpen(false);
        selectRef.current?.blur();
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [menuIsOpen]);

  // Covers every close path (selection, click outside, Escape, Tab, toggle).
  useEffect(() => {
    if (!menuIsOpen) setSearchQuery("");
  }, [menuIsOpen]);

  const handleChange = (option: SingleValue<CustomOptionType>) => {
    if (option === null) return;

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

  // Dispatches the key on react-select's hidden input so its own keydown
  // handling (and onKeyDown above) runs.
  const forwardNavKey = (event: React.KeyboardEvent<HTMLInputElement>) => {
    // SelectInstance doesn't publicly type inputRef.
    const hiddenInput = ((selectRef.current as unknown) as {
      inputRef?: HTMLInputElement | null;
    } | null)?.inputRef;
    hiddenInput?.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: event.key,
        code: event.code,
        shiftKey: event.shiftKey,
        bubbles: true,
        cancelable: true,
      })
    );
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
      backgroundColor: COLORS["core-fleet-white"],
      boxShadow: `0 0 0 1px ${COLORS["ui-fleet-black-10"]}`,
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
      <div onClick={toggleMenu} ref={wrapperRef}>
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
          // Tab closes the menu; it shouldn't also pick the highlighted option.
          tabSelectsValue={false}
          value={getValue()}
          onChange={handleChange}
          searchQuery={searchQuery}
          noOptionsMessage={() => "No items match this search criteria."}
          onKeyDown={onKeyDown}
          onChangeSearchQuery={onChangeSearchQuery}
          forwardNavKey={forwardNavKey}
        />
      </div>
    </FormField>
  );
};

export default ActivityTypeDropdown;
