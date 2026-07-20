import React, { useContext, useEffect, useRef, useState } from "react";
import Select, {
  components,
  DropdownIndicatorProps,
  OptionProps,
  SelectInstance,
  StylesConfig,
} from "react-select-5";

import { PADDING } from "styles/var/padding";
import { COLORS } from "styles/var/colors";
import classnames from "classnames";

import { IDropdownOption } from "interfaces/dropdownOption";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import DropdownOptionTooltipWrapper from "components/forms/fields/Dropdown/DropdownOptionTooltipWrapper";
import TableLayoutContext from "components/TableContainer/TableLayoutContext";

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
  variant?: "button" | "brand-button" | "small-button";
  buttonLabel?: string;
}

const getOptionBackgroundColor = (state: { isFocused: boolean }) => {
  return state.isFocused ? COLORS["ui-fleet-black-5"] : "transparent";
};

// "small-button" mirrors the bordered secondary button — see #35329
const getControlBackgroundColor = (
  variant: string | undefined,
  isFocused: boolean
) => {
  if (isFocused) return COLORS["ui-fleet-black-5"];
  return variant === "small-button" ? COLORS["ui-off-white"] : "initial";
};

const getLeftMenuAlign = (menuAlign: "right" | "left" | "default") => {
  switch (menuAlign) {
    case "right":
      return "auto";
    case "left":
      return "0";
    default:
      return "undefined";
  }
};

const getRightMenuAlign = (menuAlign: "right" | "left" | "default") => {
  switch (menuAlign) {
    case "right":
      return "0";
    case "left":
      return "auto";
    default:
      return "undefined";
  }
};

const CustomDropdownIndicator = (
  props: DropdownIndicatorProps<IDropdownOption, false>
) => {
  const { isFocused, selectProps } = props;
  const variant = (selectProps as {
    variant?: "button" | "brand-button" | "small-button";
  }).variant;

  const color =
    isFocused ||
    selectProps.menuIsOpen ||
    variant === "button" ||
    variant === "small-button"
      ? "ui-fleet-black-75"
      : "core-fleet-black";

  return (
    <components.DropdownIndicator {...props} className={baseClass}>
      <Icon
        name="chevron-down"
        color={color}
        size={variant === "small-button" ? "small" : undefined}
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
  variant,
  buttonLabel,
}: IActionsDropdownProps): JSX.Element => {
  const dropdownClassnames = classnames(baseClass, className);

  // Portal the menu only when rendered inside a TableContainer's data-table
  // block, where .data-table__wrapper's overflow-x: auto would otherwise clip
  // the menu vertically. The brand-button variant nulls out react-select's
  // Control, and MenuPortal bails when controlElement is missing — so don't
  // use brand-button inside a table cell.
  const { insideTable } = useContext(TableLayoutContext);

  // Used for brand Action button
  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const selectRef = useRef<SelectInstance<IDropdownOption, false>>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (!menuIsOpen || !wrapperRef.current) return;
      const target = event.target;
      if (!(target instanceof Node)) return;
      // Trigger button (wrapper) or portaled menu both count as "inside" —
      // since menuPortalTarget renders the menu in document.body, a contains()
      // check on wrapperRef alone would treat option clicks as outside.
      if (wrapperRef.current.contains(target)) return;
      if (
        target instanceof Element &&
        target.closest(`.${baseClass}-select__menu-portal`)
      ) {
        return;
      }
      setMenuIsOpen(false);
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [menuIsOpen]);

  const isBrandButton = variant === "brand-button";

  // CustomControl rerenders on state change, preventing arrow animation
  // Render brand button outside of CustomControl instead
  const renderBrandButton = () => (
    <Button
      type="button"
      onClick={() => setMenuIsOpen((v) => !v)}
      className={`${baseClass}__button`}
      disabled={disabled}
      aria-haspopup="listbox"
      aria-expanded={menuIsOpen}
    >
      <span>{buttonLabel || "Actions"}</span>
      <Icon
        name="chevron-down"
        color="core-fleet-white"
        className={`actions-dropdown__icon${
          menuIsOpen ? " actions-dropdown__icon--open" : ""
        }`}
      />
    </Button>
  );

  const handleChange = (newValue: IDropdownOption | null) => {
    if (newValue) {
      onChange(newValue.value.toString());
      setMenuIsOpen(false); // close menu on select
    }
  };

  const customStyles: StylesConfig<IDropdownOption, false> = {
    control: (provided, state) => ({
      ...provided,
      display: "flex",
      flexDirection: "row",
      width: "max-content",
      // Need minHeight to override default
      minHeight: variant === "small-button" ? "28px" : "32px", // Match button height
      padding: variant === "small-button" ? "4px 8px" : "8px", // Match button padding
      backgroundColor: getControlBackgroundColor(variant, state.isFocused),
      border:
        variant === "small-button"
          ? `1px solid ${COLORS["ui-fleet-black-25"]}` // Match secondary button border — see #35329
          : 0,
      boxSizing: "border-box",
      boxShadow: "none",
      cursor: "pointer",
      "&:hover": {
        background: COLORS["ui-fleet-black-5"], // Match button hover
        boxShadow: "none",
        ".actions-dropdown-select__placeholder": {
          color: COLORS["ui-fleet-black-75-over"],
        },
        ".actions-dropdown-select__indicator path": {
          stroke: COLORS["ui-fleet-black-75-over"],
        },
      },
      "&:active": {
        background:
          variant === "small-button"
            ? COLORS["ui-fleet-black-10"] // Match secondary button active — see #35329
            : COLORS["ui-fleet-black-5"], // Match button hover
        ".actions-dropdown-select__indicator path": {
          stroke: COLORS["ui-fleet-black-75-down"],
        },
      },
      // TODO: Figure out a way to apply separate &:focus-visible styling
      // Currently only relying on &:focus styling for tabbing through app
      ...(state.menuIsOpen && {
        background: COLORS["ui-fleet-black-5"], // Match button hover
        ".actions-dropdown-select__indicator svg": {
          transform: "rotate(180deg)",
          transition: "transform 0.25s ease",
        },
      }),
    }),
    placeholder: (provided, state) => ({
      ...provided,
      color:
        state.isFocused || variant === "button" || variant === "small-button"
          ? COLORS["ui-fleet-black-75"]
          : COLORS["core-fleet-black"],
      fontSize: "14px",
      fontWeight:
        variant === "button" || variant === "small-button" ? "600" : undefined,
      lineHeight: "normal",
      paddingLeft: 0,
      marginTop: "1px",
      ...(state.isDisabled && {
        filter: "grayscale(0.5)",
        opacity: 0.5,
      }),
    }),
    dropdownIndicator: (provided, state) => ({
      ...provided,
      display: "flex",
      padding: "2px",
      svg: {
        transition: "transform 0.25s ease",
      },
      ...(state.isDisabled && {
        filter: "grayscale(0.5)",
        opacity: 0.5,
      }),
    }),
    menu: (provided) => ({
      ...provided,
      backgroundColor: COLORS["core-fleet-white"],
      boxShadow: `0 2px 6px rgba(0, 0, 0, 0.1), 0 0 0 1px var(--dropdown-menu-outline, transparent)`,
      borderRadius: "4px",
      zIndex: 6,
      border: 0,
      marginTop: isBrandButton ? "20px" : "0",
      width: "auto",
      minWidth: "100%",
      position: "absolute",
      left: getLeftMenuAlign(menuAlign),
      right: getRightMenuAlign(menuAlign),
      animation: "fade-in 150ms ease-out",
    }),
    // zIndex 999 (document-portal tier) so the portaled menu clears
    // .site-nav-container and Modal — ActionsDropdown can render inside a
    // TableContainer that lives inside a modal (e.g. ScriptDetailsModal).
    menuPortal: (provided) => ({
      ...provided,
      zIndex: 999,
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
      fontSize: "13px",
      backgroundColor: getOptionBackgroundColor(state),
      whiteSpace: "nowrap",
      "&:hover": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-fleet-black-5"],
      },
      "&:active": {
        backgroundColor: state.isDisabled
          ? "transparent"
          : COLORS["ui-fleet-black-5"],
      },
      ...(state.isDisabled && {
        color: COLORS["ui-fleet-black-50"],
        fontStyle: "italic",
      }),
    }),
  };

  return (
    <div className={`${baseClass}__wrapper`} ref={wrapperRef}>
      {isBrandButton && renderBrandButton()}
      <Select<IDropdownOption, false>
        ref={selectRef}
        options={options}
        placeholder={isBrandButton ? "" : placeholder}
        onChange={handleChange}
        isDisabled={disabled}
        isSearchable={isSearchable}
        styles={customStyles}
        menuIsOpen={menuIsOpen}
        onMenuOpen={() => setMenuIsOpen(true)} // Needed abstraction for brand-action button
        onMenuClose={() => setMenuIsOpen(false)} // Needed abstraction for brand-action-button
        components={{
          DropdownIndicator: CustomDropdownIndicator,
          IndicatorSeparator: () => null,
          Option: CustomOption,
          SingleValue: () => null, // Doesn't replace placeholder text with selected text
          // Note: react-select doesn't support skipping disabled options when keyboarding through
          ...(isBrandButton && { Control: () => null }), // Remove Control entirely and renderBrandButton instead
        }}
        controlShouldRenderValue={false} // Doesn't change placeholder text to selected text
        isOptionSelected={() => false} // Hides any styling on selected option
        value={null} // Prevent an option from being selected
        className={dropdownClassnames}
        classNamePrefix={`${baseClass}-select`}
        isOptionDisabled={(option) => !!option.disabled}
        menuPlacement={menuPlacement}
        menuPortalTarget={insideTable ? document.body : undefined}
        {...{ variant }} // Allows CustomDropdownIndicator to be ui-fleet-black-75 for variant: "button"
      />
    </div>
  );
};

export default ActionsDropdown;
