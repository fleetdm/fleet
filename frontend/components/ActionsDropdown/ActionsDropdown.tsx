import React from "react";
import Select, {
  StylesConfig,
  DropdownIndicatorProps,
  OptionProps,
  components,
} from "react-select-5";

import { PADDING } from "styles/var/padding";
import { COLORS } from "styles/var/colors";
import classnames from "classnames";

import { IDropdownOption } from "interfaces/dropdownOption";

import Icon from "components/Icon";
import DropdownOptionTooltipWrapper from "components/forms/fields/Dropdown/DropdownOptionTooltipWrapper";

const baseClass = "actions-dropdown";

interface IActionsDropdownProps {
  options: IDropdownOption[];
  placeholder: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  isSearchable?: boolean;
  className?: string;
  menuAlign?: "right" | "left" | "default";
  menuPlacement?: "top" | "bottom" | "auto";
}

const getOptionBackgroundColor = (state: any) => {
  return state.isFocused ? COLORS["ui-vibrant-blue-10"] : "transparent";
};

const getLeftMenuAlign = (menuAlign: "right" | "left" | "default") => {
  switch (menuAlign) {
    case "right":
      return "auto";
    case "left":
      return "0";
    default:
      return "-12px";
  }
};

const getRightMenuAlign = (menuAlign: "right" | "left" | "default") => {
  switch (menuAlign) {
    case "right":
      return "0";
    default:
      return "undefined";
  }
};

const CustomDropdownIndicator = (
  props: DropdownIndicatorProps<any, false, any>
) => {
  const { isFocused, selectProps } = props;
  // no access to hover state here from react-select so that is done in the scss
  // file of ActionsDropdown.
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

const CustomOption: React.FC<OptionProps<IDropdownOption, false>> = (props) => {
  const { innerRef, data, isDisabled } = props;

  const optionContent = (
    <div
      className={`${baseClass}__option`}
      ref={innerRef}
      tabIndex={isDisabled ? -1 : 0} // Tabbing skipped when disabled
      aria-disabled={isDisabled}
    >
      {data.label}
      {data.helpText && (
        <span className={`${baseClass}__help-text`}>{data.helpText}</span>
      )}
    </div>
  );

  return (
    <components.Option {...props}>
      {data.tooltipContent ? (
        <DropdownOptionTooltipWrapper tipContent={data.tooltipContent}>
          {optionContent}
        </DropdownOptionTooltipWrapper>
      ) : (
        optionContent
      )}
    </components.Option>
  );
};

const ActionsDropdown = ({
  options,
  placeholder,
  onChange,
  disabled,
  isSearchable = false,
  className,
  menuAlign = "default",
  menuPlacement = "bottom",
}: IActionsDropdownProps): JSX.Element => {
  const dropdownClassnames = classnames(baseClass, className);

  const handleChange = (newValue: IDropdownOption | null) => {
    if (newValue) {
      onChange(newValue.value.toString());
    }
  };

  const customStyles: StylesConfig<IDropdownOption, false> = {
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
        ".actions-dropdown-select__placeholder": {
          color: COLORS["core-vibrant-blue-over"],
        },
        ".actions-dropdown-select__indicator path": {
          stroke: COLORS["core-vibrant-blue-over"],
        },
      },
      "&:active .actions-dropdown-select__indicator path": {
        stroke: COLORS["core-vibrant-blue-down"],
      },
      // TODO: Figure out a way to apply separate &:focus-visible styling
      // Currently only relying on &:focus styling for tabbing through app
      ...(state.menuIsOpen && {
        ".actions-dropdown-select__indicator svg": {
          transform: "rotate(180deg)",
          transition: "transform 0.25s ease",
        },
      }),
    }),
    placeholder: (provided, state) => ({
      ...provided,
      color: state.isFocused
        ? COLORS["core-vibrant-blue"]
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
      border: 0,
      margin: 0,
      width: "auto",
      minWidth: "100%",
      position: "absolute",
      left: getLeftMenuAlign(menuAlign),
      right: getRightMenuAlign(menuAlign),
      animation: "fade-in 150ms ease-out",
    }),
    menuList: (provided) => ({
      ...provided,
      padding: PADDING["pad-small"],
      maxHeight: "initial", // Override react-select default height of 300px to avoid scrollbar on hostactionsdropdown
    }),
    valueContainer: (provided) => ({
      ...provided,
      padding: 0,
    }),
    option: (provided, state) => ({
      ...provided,
      padding: "10px 8px",
      borderRadius: "4px",
      fontSize: "14px",
      backgroundColor: getOptionBackgroundColor(state),
      whiteSpace: "nowrap",
      "&:hover": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-vibrant-blue-10"],
      },
      "&:active": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-vibrant-blue-25"],
      },
      ...(state.isDisabled && {
        color: COLORS["ui-fleet-black-50"],
        fontStyle: "italic",
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
        isSearchable={isSearchable}
        styles={customStyles}
        components={{
          DropdownIndicator: CustomDropdownIndicator,
          IndicatorSeparator: () => null,
          Option: CustomOption,
          SingleValue: () => null, // Doesn't replace placeholder text with selected text
          // Note: react-select doesn't support skipping disabled options when keyboarding through
        }}
        controlShouldRenderValue={false} // Doesn't change placeholder text to selected text
        isOptionSelected={() => false} // Hides any styling on selected option
        value={null} // Prevent an option from being selected
        className={dropdownClassnames}
        classNamePrefix={`${baseClass}-select`}
        isOptionDisabled={(option) => !!option.disabled}
        menuPlacement={menuPlacement}
      />
    </div>
  );
};

export default ActionsDropdown;
