import React from "react";
import Select, {
  StylesConfig,
  DropdownIndicatorProps,
  components,
} from "react-select-5";
import { PADDING } from "styles/var/padding";
import { COLORS } from "styles/var/colors";

import { IDropdownOption } from "interfaces/dropdownOption";

import Icon from "components/Icon";

const baseClass = "dropdown-cell";

interface IDropdownCellProps {
  options: IDropdownOption[];
  placeholder: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

const getOptionBackgroundColor = (state: any) => {
  if (state.isSelected || state.isFocused) {
    return COLORS["ui-vibrant-blue-25"];
  }
  if (state.isFocused) {
    return COLORS["ui-vibrant-blue-10"];
  }
  return "transparent";
};

const CustomDropdownIndicator = (
  props: DropdownIndicatorProps<any, false, any>
) => {
  const { isFocused, selectProps } = props;
  // no access to hover state here from react-select so that is done in the scss
  // file of DropdownCell.
  const color =
    isFocused || selectProps.menuIsOpen
      ? "core-fleet-blue"
      : "core-fleet-black";

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

const DropdownCell = ({
  options,
  placeholder,
  onChange,
  disabled,
}: IDropdownCellProps): JSX.Element => {
  const handleChange = (newValue: IDropdownOption | null) => {
    if (newValue) {
      onChange(newValue.value.toString());
    }
  };

  const customStyles: StylesConfig<IDropdownOption, false> = {
    container: (provided) => ({
      ...provided,
      width: "80px",
    }),
    control: (provided, state) => ({
      ...provided,
      display: "flex",
      flexDirection: "row",
      width: "max-content",
      padding: "8px 0",
      backgroundColor: "initial",
      border: 0,
      boxShadow: "none",
      cursor: "pointer",
      "&:hover": {
        boxShadow: "none",
        ".dropdown-cell-select__placeholder": {
          color: COLORS["core-vibrant-blue-over"],
        },
        ".dropdown-cell-select__indicator path": {
          stroke: COLORS["core-vibrant-blue-over"],
        },
      },
      "&:active .dropdown-cell-select__indicator path": {
        stroke: COLORS["core-vibrant-blue-down"],
      },
      ...(state.menuIsOpen && {
        ".dropdown-cell-select__indicator svg": {
          transform: "rotate(180deg)",
          transition: "transform 0.25s ease",
        },
      }),
    }),
    placeholder: (provided, state) => ({
      ...provided,
      color: state.isFocused
        ? COLORS["core-fleet-blue"]
        : COLORS["core-fleet-black"],
      fontSize: "14px",
      lineHeight: "normal",
      paddingLeft: 0,
      marginTop: "1px",
    }),
    dropdownIndicator: (provided) => ({
      ...provided,
      display: "flex",
      padding: "2px",
      svg: {
        transition: "transform 0.25s ease",
      },
    }),
    menu: (provided) => ({
      ...provided,
      boxShadow: "0 2px 6px rgba(0, 0, 0, 0.1)",
      borderRadius: "4px",
      zIndex: 6,
      overflow: "hidden",
      border: 0,
      marginTop: 0,
      minWidth: "158px",
      maxHeight: "220px",
      position: "absolute",
      left: "-12px",
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (provided) => ({
      ...provided,
      padding: PADDING["pad-small"],
    }),
    valueContainer: (provided) => ({
      ...provided,
      padding: 0,
    }),
    option: (provided, state) => ({
      ...provided,
      padding: "10px 8px",
      cursor: "pointer",
      backgroundColor: getOptionBackgroundColor(state),
      "&:hover": {
        cursor: "pointer",
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-vibrant-blue-10"],
      },
      "&:active": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-vibrant-blue-25"],
      },
      ...(state.isSelected && {
        backgroundColor: COLORS["ui-vibrant-blue-25"],
      }),
      ...(state.isFocused && {
        backgroundColor: COLORS["ui-vibrant-blue-10"],
      }),
      ...(state.isDisabled && {
        "&:active": {
          backgroundColor: "transparent",
        },
      }),
    }),
  };

  return (
    <div className={baseClass}>
      <Select<IDropdownOption, false>
        options={options}
        placeholder={placeholder}
        onChange={handleChange}
        isDisabled={disabled}
        isSearchable={false}
        styles={customStyles}
        components={{
          DropdownIndicator: CustomDropdownIndicator,
          IndicatorSeparator: () => null,
        }}
        className={`${baseClass}-select`}
        classNamePrefix={`${baseClass}-select`}
        tabIndex={0}
        isOptionDisabled={(option) => !!option.disabled}
      />
    </div>
  );
};

export default DropdownCell;
