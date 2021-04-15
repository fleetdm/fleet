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
    disabled: PropTypes.bool,
    error: PropTypes.string,
    label: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    labelClassName: PropTypes.string,
    multi: PropTypes.bool,
    name: PropTypes.string,
    onChange: PropTypes.func,
    options: PropTypes.arrayOf(dropdownOptionInterface).isRequired,
    placeholder: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    value: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    wrapperClassName: PropTypes.string,
  };

  static defaultProps = {
    onChange: noop,
    clearable: false,
    disabled: false,
    multi: false,
    name: "targets",
    placeholder: "Select One...",
  };

  handleChange = (selected) => {
    const { multi, onChange } = this.props;

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
    const { handleChange, renderOption } = this;
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
          name={`${name}-select`}
          onChange={handleChange}
          options={options}
          optionRenderer={renderOption}
          placeholder={placeholder}
          value={value}
        />
      </FormField>
    );
  }
}

export default Dropdown;
