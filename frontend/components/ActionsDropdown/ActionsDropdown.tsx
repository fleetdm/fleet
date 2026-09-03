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
import { IconNames } from "components/icons";

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
  /** Mirrors Fleet's Primary/Secondary/Subdued button styles — see #35329.
   *  Default: "subdued" */
  variant?: "primary" | "secondary" | "subdued";
  buttonLabel?: string;
  /** Renders an icon-only trigger button (e.g. a settings gear) instead of the
   * text control, following the same Control-replacement mechanism as the
   * primary variant. `placeholder` becomes the button's accessible label. */
  triggerIcon?: IconNames;
}

const getOptionBackgroundColor = (state: { isFocused: boolean }) => {
  return state.isFocused ? COLORS["ui-fleet-black-5"] : "transparent";
};

const getControlBackgroundColor = (variant: string | undefined) => {
  return variant === "secondary" ? COLORS["ui-off-white"] : "initial";
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
    variant?: "primary" | "secondary" | "subdued";
  }).variant;

  const color =
    isFocused ||
    selectProps.menuIsOpen ||
    variant === "subdued" ||
    variant === "secondary"
      ? "ui-fleet-black-75"
      : "core-fleet-black";

  return (
    <components.DropdownIndicator {...props} className={baseClass}>
      <Icon
        name="chevron-down"
        color={color}
        size={variant === "secondary" ? "small" : undefined}
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
      data-testid="dropdown-option"
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
  menuPlacement,
  variant = "subdued",
  buttonLabel,
  triggerIcon,
}: IActionsDropdownProps): JSX.Element => {
  const dropdownClassnames = classnames(baseClass, className);

  // Portal the menu only when rendered inside a TableContainer's data-table
  // block, where .data-table__wrapper's overflow-x: auto would otherwise clip
  // the menu vertically. The primary variant nulls out react-select's
  // Control, and MenuPortal bails when controlElement is missing — so don't
  // use primary inside a table cell.
  const { insideTable } = useContext(TableLayoutContext);

  // Used for the primary Action button
  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const selectRef = useRef<SelectInstance<IDropdownOption, false>>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // react-select's hidden input always matches :focus-visible, even on a
  // mouse click (browsers treat text inputs specially), so CSS alone can't
  // tell a Tab-focus apart from a click. Track the last input method
  // ourselves — same approach as UserMenu.tsx — so the focus ring only
  // shows up for keyboard tabbing.
  const [isKeyboardFocus, setIsKeyboardFocus] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Tab") {
        setIsKeyboardFocus(true);
      }
    };

    const handleMouseDown = () => {
      setIsKeyboardFocus(false);
    };

    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("mousedown", handleMouseDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("mousedown", handleMouseDown);
    };
  }, []);

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

  const isPrimary = variant === "primary";
  const hasIconTrigger = !!triggerIcon;

  const toggleMenu = () => setMenuIsOpen((isOpen) => !isOpen);

  // Same Control-replacement approach as the primary variant: the trigger is a
  // real Button so it gets Fleet's button styles — including the square
  // icon-only treatment for a lone Icon child — and focus handling for free.
  const renderIconTriggerButton = () => (
    <Button
      type="button"
      variant="secondary"
      onClick={toggleMenu}
      // Button also invokes onClick from its own Enter-keydown handler, and
      // the native <button> fires a click on Enter — a toggle would run twice
      // and the menu would stay closed. preventDefault suppresses the native
      // click so Enter toggles exactly once.
      customOnKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          toggleMenu();
        }
      }}
      className={`${baseClass}__icon-trigger`}
      disabled={disabled}
      ariaHasPopup="listbox"
      ariaExpanded={menuIsOpen}
      ariaLabel={placeholder}
    >
      {triggerIcon && <Icon name={triggerIcon} />}
    </Button>
  );

  // CustomControl rerenders on state change, preventing arrow animation
  // Render primary button outside of CustomControl instead
  const renderPrimaryButton = () => (
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
      gap: "8px",
      width: "max-content",
      // Need minHeight to override default
      minHeight: variant === "secondary" ? "28px" : "32px", // Match button height
      padding: variant === "secondary" ? "4px 8px" : "8px", // Match button padding
      backgroundColor: getControlBackgroundColor(variant),
      border:
        variant === "secondary"
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
          variant === "secondary"
            ? COLORS["ui-fleet-black-10"] // Match secondary button active — see #35329
            : COLORS["ui-fleet-black-5"], // Match button hover
        ".actions-dropdown-select__indicator path": {
          stroke: COLORS["ui-fleet-black-75-down"],
        },
      },
      ...(state.menuIsOpen && {
        background: COLORS["ui-fleet-black-5"], // Match button hover
        ".actions-dropdown-select__indicator svg": {
          transform: "rotate(180deg)",
          transition: "transform 0.25s ease",
        },
      }),
      // Same ring Button's :focus-visible draws — only for keyboard tabbing,
      // never a mouse click (see isKeyboardFocus above).
      //
      // Drawn INSIDE the control (via outline + negative offset) rather than
      // as an outset box-shadow. An outset shadow renders as a full-contrast
      // 1px halo on the page background, and for the bordered `secondary`
      // variant it stacks against the existing 1px grey border → reads as a
      // ~2px band. Sitting inside the box overlays the border pixel and
      // matches Button's own `::after` focus ring visual weight.
      ...(state.isFocused &&
        isKeyboardFocus && {
          outline: `1px solid ${COLORS["core-fleet-black"]}`,
          outlineOffset: "-1px",
        }),
    }),
    placeholder: (provided, state) => ({
      ...provided,
      color:
        state.isFocused || variant === "subdued" || variant === "secondary"
          ? COLORS["ui-fleet-black-75"]
          : COLORS["core-fleet-black"],
      fontSize: "14px",
      fontWeight:
        variant === "subdued" || variant === "secondary" ? "600" : undefined,
      lineHeight: "normal",
      paddingLeft: 0,
      margin: 0,
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
    // With the Control nulled out, react-select's container collapses to 0x0
    // and, as a flex item, sits vertically centered against the trigger button
    // — so the menu's `top: 100%` lands mid-button.
    // Stretch the container over the button instead, so the menu hangs off the button's
    // edge in either placement.
    // Pointer events are off so the overlay can't swallow the trigger's clicks;
    // the menu turns them back on.
    container: (provided) => ({
      ...provided,
      ...(hasIconTrigger && {
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
      }),
    }),
    menu: (provided) => ({
      ...provided,
      backgroundColor: COLORS["core-fleet-white"],
      boxShadow: `0 2px 6px rgba(0, 0, 0, 0.1), 0 0 0 1px ${COLORS["ui-fleet-black-10"]}`,
      borderRadius: "4px",
      zIndex: 6,
      border: 0,
      marginTop: isPrimary ? "20px" : "0",
      marginBottom: isPrimary ? "20px" : "0",
      width: "auto",
      minWidth: "100%",
      position: "absolute",
      left: getLeftMenuAlign(menuAlign),
      right: getRightMenuAlign(menuAlign),
      animation: "fade-in 150ms ease-out",
      ...(hasIconTrigger && {
        marginTop: "1px",
        marginBottom: "1px",
        pointerEvents: "auto",
      }),
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
      // Match DropdownWrapper's option cursor treatment.
      cursor: state.isDisabled ? "not-allowed" : "pointer",
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

  const wrapperClassnames = classnames(`${baseClass}__wrapper`, {
    [`${baseClass}__wrapper--icon-trigger`]: hasIconTrigger,
  });

  return (
    <div className={wrapperClassnames} ref={wrapperRef}>
      {isPrimary && renderPrimaryButton()}
      {hasIconTrigger && renderIconTriggerButton()}
      <Select<IDropdownOption, false>
        ref={selectRef}
        options={options}
        placeholder={isPrimary || hasIconTrigger ? "" : placeholder}
        onChange={handleChange}
        isDisabled={disabled}
        isSearchable={isSearchable}
        styles={customStyles}
        menuIsOpen={menuIsOpen}
        onMenuOpen={() => setMenuIsOpen(true)} // Needed abstraction for the primary Action button
        onMenuClose={() => setMenuIsOpen(false)} // Needed abstraction for the primary Action button
        components={{
          DropdownIndicator: CustomDropdownIndicator,
          IndicatorSeparator: () => null,
          Option: CustomOption,
          SingleValue: () => null, // Doesn't replace placeholder text with selected text
          // Note: react-select doesn't support skipping disabled options when keyboarding through
          ...((isPrimary || hasIconTrigger) && { Control: () => null }), // Remove Control entirely and render a custom trigger button instead
        }}
        controlShouldRenderValue={false} // Doesn't change placeholder text to selected text
        isOptionSelected={() => false} // Hides any styling on selected option
        value={null} // Prevent an option from being selected
        className={dropdownClassnames}
        classNamePrefix={`${baseClass}-select`}
        isOptionDisabled={(option) => !!option.disabled}
        menuPlacement={menuPlacement ?? (insideTable ? "auto" : "bottom")}
        menuPortalTarget={insideTable ? document.body : undefined}
        {...{ variant }} // Allows CustomDropdownIndicator to be ui-fleet-black-75 for variant: "subdued"
      />
    </div>
  );
};

export default ActionsDropdown;
