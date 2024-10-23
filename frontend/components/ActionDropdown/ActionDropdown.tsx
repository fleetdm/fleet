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

const baseClass = "action-dropdown";

interface IActionDropdownProps {
  options: IDropdownOption[];
  placeholder: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  isSearchable?: boolean;
  className?: string;
  menuAlign?: "right" | "default";
}

const getOptionBackgroundColor = (state: any) => {
  return state.isSelected || state.isFocused
    ? COLORS["ui-vibrant-blue-10"]
    : "transparent";
};

const CustomDropdownIndicator = (
  props: DropdownIndicatorProps<any, false, any>
) => {
  const { isFocused, selectProps } = props;
  // no access to hover state here from react-select so that is done in the scss
  // file of ActionDropdown.
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
  const { innerProps, innerRef, data, isDisabled } = props;

  const optionContent = (
    <div
      className={`${baseClass}__option`}
      ref={innerRef}
      {...innerProps}
      tabIndex={isDisabled ? -1 : 0} // Tabbing skipped when disabled
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

const ActionDropdown = ({
  options,
  placeholder,
  onChange,
  disabled,
  isSearchable = false,
  className,
  menuAlign = "default",
}: IActionDropdownProps): JSX.Element => {
  const dropdownClassnames = classnames(baseClass, className);

  const [isKeyboardFocused, setIsKeyboardFocused] = React.useState(false);

  const handleChange = (newValue: IDropdownOption | null) => {
    if (newValue) {
      onChange(newValue.value.toString());
    }
  };

  const handleFocus = (event: React.FocusEvent) => {
    // Check if the focus event was triggered by keyboard
    if (event.target === event.currentTarget) {
      setIsKeyboardFocused(true);
    }
  };

  const handleBlur = () => {
    setIsKeyboardFocused(false);
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
        ".action-dropdown-select__placeholder": {
          color: COLORS["core-vibrant-blue-over"],
        },
        ".action-dropdown-select__indicator path": {
          stroke: COLORS["core-vibrant-blue-over"],
        },
      },
      "&:active .action-dropdown-select__indicator path": {
        stroke: COLORS["core-vibrant-blue-down"],
      },
      // TODO: Figure out a way to apply separate &:focus-visible styling
      // Currently only relying on &:focus styling for tabbing through app
      ...(state.menuIsOpen && {
        ".action-dropdown-select__indicator svg": {
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
      left: menuAlign === "default" ? "-12px" : "auto",
      right: menuAlign === "right" ? 0 : undefined,
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
      fontSize: "14px",
      backgroundColor: getOptionBackgroundColor(state),
      "&:hover": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-vibrant-blue-10"],
      },
      "&:active": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-vibrant-blue-10"],
      },
      ...(state.isDisabled && {
        color: COLORS["ui-fleet-black-50"],
        fontStyle: "italic",
        // pointerEvents: "none", // Prevents any mouse interaction
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
          // TODO: Figure out how to skip disabled options when keyboarding through options
        }}
        controlShouldRenderValue={false} // Doesn't change placeholder text to selected text
        isOptionSelected={() => false} // Hides any styling on selected option
        className={dropdownClassnames}
        classNamePrefix={`${baseClass}-select`}
        isOptionDisabled={(option) => !!option.disabled}
      />
    </div>
  );
};

export default ActionDropdown;
