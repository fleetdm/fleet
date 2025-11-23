import React, { useRef } from "react";
import Select, {
  components,
  MenuListProps,
  SingleValue,
  GroupBase,
} from "react-select-5";

import { ACTIVITY_DISPLAY_NAME_MAP, ActivityType } from "interfaces/activity";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
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
    onClickSearchInput?.(event);
    inputRef.current?.focus();
    event.stopPropagation();
  };

  return (
    <components.MenuList {...props}>
      <div className={`${baseClass}__field`}>
        <input
          className={`${baseClass}__input`}
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

const TYPE_FILTER_OPTIONS: CustomOptionType[] = Object.values(ActivityType)
  .map((type) => ({
    label: ACTIVITY_DISPLAY_NAME_MAP[type],
    value: type,
  }))
  .sort((a, b) => a.label.localeCompare(b.label));

TYPE_FILTER_OPTIONS.unshift({
  label: "All types",
  value: "all",
});

interface IActivityTypeDropdownProps {
  value: string;
  onSelect: (value: string) => void;
}

const ActivityTypeDropdown = ({
  value,
  onSelect,
}: IActivityTypeDropdownProps) => {
  const [searchQuery, setSearchQuery] = React.useState("");

  const handleChange = (option: SingleValue<CustomOptionType>) => {
    onSelect(option ? option.value : "all");
  };

  const onInputChange = (val: string, actionMeta: any) => {
    console.log("Input Changed", val, actionMeta);
  };

  const getValue = () => {
    return (
      TYPE_FILTER_OPTIONS.find((option) => option.value === value) ||
      TYPE_FILTER_OPTIONS[0]
    );
  };

  return (
    <Select<CustomOptionType, false>
      className={baseClass}
      // styles={customStyles}
      options={TYPE_FILTER_OPTIONS}
      components={{
        MenuList: CustomMenuList,
        // Option: CustomOption,
        // DropdownIndicator: CustomDropdownIndicator,
        IndicatorSeparator: () => null,
        // ValueContainer,
      }}
      isSearchable={false}
      value={getValue()}
      onChange={handleChange}
      searchQuery={searchQuery}
      // onInputChange={onInputChange}
      noOptionsMessage={() => "No results found"}
    />
  );
};

export default ActivityTypeDropdown;
