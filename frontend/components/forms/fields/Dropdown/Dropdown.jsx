import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { noop, pick } from "lodash";
import Select from "react-select";

import dropdownOptionInterface from "interfaces/dropdownOption";
import FormField from "components/forms/FormField";

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
    placeholder: "Select One...",
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
    const { multi, onChange, clearable } = this.props;

    if (clearable && selected === null) {
      onChange(null);
    } else if (multi) {
      onChange(selected.map((obj) => obj.value).join(","));
    } else {
      onChange(selected.value);
    }
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
    return (
      <div className={`${baseClass}__option`}>
        {option.label}
        {option.helpText && (
          <span className={`${baseClass}__help-text`}>{option.helpText}</span>
        )}
      </div>
    );
  };

  render() {
    const { handleChange, renderOption, onMenuOpen, onMenuClose } = this;
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
    } = this.props;

    const formFieldProps = pick(this.props, ["hint", "label", "error", "name"]);
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
        />
      </FormField>
    );
  }
}

export default Dropdown;
