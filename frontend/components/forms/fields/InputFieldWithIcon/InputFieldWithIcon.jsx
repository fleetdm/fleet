import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import Icon from "components/Icon/Icon";
import FleetIcon from "components/icons/FleetIcon";
import TooltipWrapper from "components/TooltipWrapper";
import InputField from "../InputField";

const baseClass = "input-icon-field";

class InputFieldWithIcon extends InputField {
  static propTypes = {
    autofocus: PropTypes.bool,
    error: PropTypes.string,
    hint: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    iconName: PropTypes.string,
    iconSvg: PropTypes.string,
    label: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    tabIndex: PropTypes.number,
    type: PropTypes.string,
    className: PropTypes.string,
    disabled: PropTypes.bool,
    iconPosition: PropTypes.oneOf(["start", "end"]),
    inputOptions: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    tooltip: PropTypes.string,
  };

  renderHeading = () => {
    const { error, placeholder, name, tooltip } = this.props;
    const label = this.props.label ?? placeholder;

    const labelClasses = classnames(`${baseClass}__label`, {
      [`${baseClass}__errors`]: !!error,
    });

    return (
      <label
        htmlFor={name}
        className={labelClasses}
        data-has-tooltip={!!tooltip}
      >
        {tooltip && !error ? (
          <TooltipWrapper position="top" tipContent={tooltip}>
            {label}
          </TooltipWrapper>
        ) : (
          <>{error || label}</>
        )}
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
      iconSvg,
      name,
      placeholder,
      tabIndex,
      type,
      value,
      disabled,
      iconPosition,
      inputOptions,
    } = this.props;
    const { onInputChange, renderHint } = this;

    const wrapperClasses = classnames(baseClass, {
      [`${baseClass}--icon-start`]: iconPosition && iconPosition === "start",
    });

    const inputClasses = classnames(
      `${baseClass}__input`,
      "input-with-icon",
      className,
      { [`${baseClass}__input--error`]: error },
      { [`${baseClass}__input--password`]: type === "password" && value },
      {
        [`${baseClass}__input--icon-start`]:
          iconPosition && iconPosition === "start",
      }
    );

    const iconClasses = classnames(
      `${baseClass}__icon`,
      { [`${baseClass}__icon--error`]: error },
      { [`${baseClass}__icon--active`]: value }
    );

    return (
      <div className={wrapperClasses}>
        {this.props.label && this.renderHeading()}
        {iconSvg && <Icon name={iconSvg} className={iconClasses} />}
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
          {...inputOptions}
        />
        {iconName && <FleetIcon name={iconName} className={iconClasses} />}
        {renderHint()}
      </div>
    );
  }
}

export default InputFieldWithIcon;
