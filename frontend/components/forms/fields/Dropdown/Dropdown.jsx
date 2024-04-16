import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { noop, pick } from "lodash";
import Select from "react-select";

import dropdownOptionInterface from "interfaces/dropdownOption";
import FormField from "components/forms/FormField";
import Icon from "components/Icon";
import DisabledOptionTooltipWrapper from "./DisabledOptionTooltipWrapper";

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
    value: PropTypes.oneOfType([
      PropTypes.array,
      PropTypes.string,
      PropTypes.number,
    ]),
    wrapperClassName: PropTypes.string,
    parseTarget: PropTypes.bool,
    tooltip: PropTypes.string,
    autoFocus: PropTypes.bool,
    /** Includes styled filter icon */
    tableFilterDropdown: PropTypes.bool,
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
    tableFilterDropdown: false,
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
      // Returns both name and value
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
    if (option.disabledTooltipContent) {
      return (
        <DisabledOptionTooltipWrapper
          tipContent={option.disabledTooltipContent}
        >
          <div className={`${baseClass}__option`}>
            {option.label}
            {option.helpText && (
              <span className={`${baseClass}__help-text`}>
                {option.helpText}
              </span>
            )}
          </div>
        </DisabledOptionTooltipWrapper>
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

  // Adds styled filter icon to dropdown
  renderCustomTableFilter = () => {
    const { options, value } = this.props;
    const customLabel = options
      .filter((option) => option.value === value)
      .map((option) => option.label);

    return (
      <div className={`${baseClass}__custom-value`}>
        <Icon name="filter" className={`${baseClass}__icon`} />
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
      renderCustomTableFilter,
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
      tableFilterDropdown,
    } = this.props;

    const formFieldProps = pick(this.props, [
      "helpText",
      "label",
      "error",
      "name",
      "tooltip",
    ]);
    const selectClasses = classnames(className, `${baseClass}__select`, {
      [`${baseClass}__select--error`]: error,
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
          valueComponent={
            tableFilterDropdown ? renderCustomTableFilter : undefined
          }
        />
      </FormField>
    );
  }
}

export default Dropdown;
