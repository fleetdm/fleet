import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { noop, pick } from "lodash";
import Select from "react-select";

import { COLORS } from "styles/var/colors";
import dropdownOptionInterface from "interfaces/dropdownOption";
import FormField from "components/forms/FormField";
import Icon from "components/Icon";
import DropdownOptionTooltipWrapper from "./DropdownOptionTooltipWrapper";

const baseClass = "dropdown";

class Dropdown extends Component {
  static propTypes = {
    className: PropTypes.string,
    clearable: PropTypes.bool,
    searchable: PropTypes.bool,
    disabled: PropTypes.bool,
    error: PropTypes.string,
    label: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    labelClassName: PropTypes.string,
    multi: PropTypes.bool,
    name: PropTypes.string,
    onChange: PropTypes.func,
    onOpen: PropTypes.func,
    onClose: PropTypes.func,
    options: PropTypes.arrayOf(dropdownOptionInterface).isRequired,
    placeholder: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    /**
     value must correspond to the value of a dropdown option to render
     e.g. with options:

     [
       {
       label: "Display name",
       value: 1,  <– the id of the thing
       }
     ]

     set value to 1, not "Display name"
    */
    value: PropTypes.oneOfType([
      PropTypes.array,
      PropTypes.string,
      PropTypes.number,
    ]),
    wrapperClassName: PropTypes.string,
    parseTarget: PropTypes.bool,
    tooltip: PropTypes.string,
    autoFocus: PropTypes.bool,
    /** Includes styled icon */
    iconName: PropTypes.string,
    helpText: PropTypes.oneOfType([
      PropTypes.string,
      PropTypes.arrayOf(PropTypes.string),
      PropTypes.object,
    ]),
  };

  static defaultProps = {
    onChange: noop,
    onOpen: noop,
    onClose: noop,
    clearable: false,
    searchable: true,
    disabled: false,
    multi: false,
    name: "targets",
    placeholder: "Select one...", // if value undefined
    parseTarget: false,
    tooltip: "",
    autoFocus: false,
    iconName: "",
  };

  onMenuOpen = () => {
    const { onOpen } = this.props;
    onOpen();
  };

  onMenuClose = () => {
    const { onClose } = this.props;
    onClose();
  };

  handleChange = (selected) => {
    const { multi, onChange, clearable, name, parseTarget } = this.props;

    if (parseTarget) {
      // Returns both name of the Dropdown and value of the selected option
      return onChange({ value: selected.value, name });
    }

    if (clearable && selected === null) {
      return onChange(null);
    }

    if (multi) {
      return onChange(selected.map((obj) => obj.value).join(","));
    }

    return onChange(selected.value);
  };

  renderLabel = () => {
    const { error, label, labelClassName, name } = this.props;
    const labelWrapperClasses = classnames(
      `${baseClass}__label`,
      labelClassName,
      { [`${baseClass}__label--error`]: error }
    );

    if (!label) {
      return false;
    }

    return (
      <label className={labelWrapperClasses} htmlFor={name}>
        {error || label}
      </label>
    );
  };

  renderOption = (option) => {
    if (option.tooltipContent) {
      return (
        <DropdownOptionTooltipWrapper tipContent={option.tooltipContent}>
          <div className={`${baseClass}__option`}>
            {option.label}
            {option.helpText && (
              <span className={`${baseClass}__help-text`}>
                {option.helpText}
              </span>
            )}
          </div>
        </DropdownOptionTooltipWrapper>
      );
    }
    return (
      <div className={`${baseClass}__option`}>
        {option.label}
        {option.helpText && (
          <span className={`${baseClass}__help-text`}>{option.helpText}</span>
        )}
      </div>
    );
  };

  renderCustomDropdownArrow = () => {
    return (
      <div className={`${baseClass}__custom-arrow`}>
        <Icon name="chevron-down" className={`${baseClass}__icon`} />
      </div>
    );
  };

  // Adds styled icon to dropdown
  renderWithIcon = () => {
    const { options, value, iconName } = this.props;
    const customLabel = options
      .filter((option) => option.value === value)
      .map((option) => option.label);

    return (
      <div className={`${baseClass}__custom-value`}>
        <Icon name={iconName} className={`${baseClass}__icon`} />
        <div className={`${baseClass}__custom-value-label`}>{customLabel}</div>
      </div>
    );
  };

  render() {
    const {
      handleChange,
      renderOption,
      onMenuOpen,
      onMenuClose,
      renderCustomDropdownArrow,
      renderWithIcon,
    } = this;
    const {
      error,
      className,
      clearable,
      disabled,
      multi,
      name,
      options,
      placeholder,
      value,
      wrapperClassName,
      searchable,
      autoFocus,
      iconName,
    } = this.props;

    const formFieldProps = pick(this.props, [
      "helpText",
      "label",
      "error",
      "name",
      "tooltip",
      "disabled",
    ]);
    const selectClasses = classnames(className, `${baseClass}__select`, {
      [`${baseClass}__select--error`]: error,
      [`${baseClass}__select--disabled`]: disabled,
    });

    return (
      <FormField
        {...formFieldProps}
        type="dropdown"
        className={wrapperClassName}
      >
        <Select
          className={selectClasses}
          clearable={clearable}
          disabled={disabled}
          multi={multi}
          searchable={searchable}
          name={`${name}-select`}
          onChange={handleChange}
          options={options}
          optionRenderer={renderOption}
          placeholder={placeholder}
          value={value}
          onOpen={onMenuOpen}
          onClose={onMenuClose}
          autoFocus={autoFocus}
          arrowRenderer={renderCustomDropdownArrow}
          valueComponent={iconName ? renderWithIcon : undefined}
          tabIndex={disabled ? -1 : 0} // Ensures disabled dropdown has no keyboard accessibility
        />
      </FormField>
    );
  }
}

export default Dropdown;
