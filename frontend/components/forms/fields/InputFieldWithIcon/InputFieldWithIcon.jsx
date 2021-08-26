import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import FleetIcon from "components/icons/FleetIcon";
import InputField from "../InputField";

const baseClass = "input-icon-field";

class InputFieldWithIcon extends InputField {
  static propTypes = {
    autofocus: PropTypes.bool,
    error: PropTypes.string,
    hint: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    iconName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    tabIndex: PropTypes.number,
    type: PropTypes.string,
    className: PropTypes.string,
    disabled: PropTypes.bool,
  };

  renderHeading = () => {
    const { error, placeholder, name } = this.props;

    const labelClasses = classnames(`${baseClass}__label`);

    if (error) {
      return <div className={`${baseClass}__errors`}>{error}</div>;
    }

    return (
      <label htmlFor={name} className={labelClasses}>
        {placeholder}
      </label>
    );
  };

  renderHint = () => {
    const { hint } = this.props;

    if (hint) {
      return <span className={`${baseClass}__hint`}>{hint}</span>;
    }

    return false;
  };

  render() {
    const {
      className,
      error,
      iconName,
      name,
      placeholder,
      tabIndex,
      type,
      value,
      disabled,
    } = this.props;
    const { onInputChange, renderHint } = this;

    const inputClasses = classnames(
      `${baseClass}__input`,
      "input-with-icon",
      className,
      { [`${baseClass}__input--error`]: error },
      { [`${baseClass}__input--password`]: type === "password" && value }
    );

    const iconClasses = classnames(
      `${baseClass}__icon`,
      { [`${baseClass}__icon--error`]: error },
      { [`${baseClass}__icon--active`]: value }
    );

    return (
      <div className={baseClass}>
        {this.renderHeading()}
        <input
          id={name}
          name={name}
          onChange={onInputChange}
          className={inputClasses}
          placeholder={placeholder}
          ref={(r) => {
            this.input = r;
          }}
          tabIndex={tabIndex}
          type={type}
          value={value}
          disabled={disabled}
        />
        {iconName && <FleetIcon name={iconName} className={iconClasses} />}
        {renderHint()}
      </div>
    );
  }
}

export default InputFieldWithIcon;
