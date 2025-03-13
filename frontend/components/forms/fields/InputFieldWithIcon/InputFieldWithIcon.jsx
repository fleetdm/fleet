import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import { ICON_MAP } from "components/icons";
import Icon from "components/Icon/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import InputField from "../InputField";

const baseClass = "input-icon-field";

class InputFieldWithIcon extends InputField {
  static propTypes = {
    autofocus: PropTypes.bool,
    error: PropTypes.string,
    helpText: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    iconSvg: PropTypes.oneOf(Object.keys(ICON_MAP)),
    label: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    onClick: PropTypes.func,
    clearButton: PropTypes.func,
    placeholder: PropTypes.string,
    tabIndex: PropTypes.number,
    type: PropTypes.string,
    className: PropTypes.string,
    disabled: PropTypes.bool,
    inputOptions: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    tooltip: PropTypes.string,
    ignore1Password: PropTypes.bool,
  };

  renderHeading = () => {
    const { error, placeholder, name, tooltip, disabled } = this.props;
    const label = this.props.label ?? placeholder;

    const labelClasses = classnames(`${baseClass}__label`, {
      [`${baseClass}__errors`]: !!error,
      [`${baseClass}__label--disabled`]: disabled,
    });

    return (
      <label
        htmlFor={name}
        className={labelClasses}
        data-has-tooltip={!!tooltip}
      >
        {tooltip && !error ? (
          <TooltipWrapper position="bottom-start" tipContent={tooltip}>
            {label}
          </TooltipWrapper>
        ) : (
          <>{error || label}</>
        )}
      </label>
    );
  };

  renderHelpText = () => {
    const { helpText } = this.props;

    if (helpText) {
      return (
        <span className={`${baseClass}__help-text form-field__help-text`}>
          {helpText}
        </span>
      );
    }

    return false;
  };

  render() {
    const {
      className,
      error,
      iconSvg,
      name,
      placeholder,
      tabIndex,
      type,
      value,
      disabled,
      inputOptions,
      ignore1Password,
      onClick,
      onChange,
      clearButton,
    } = this.props;
    const { onInputChange, renderHelpText } = this;

    const wrapperClasses = classnames(baseClass, "form-field");

    const inputClasses = classnames(
      `${baseClass}__input`,
      className,
      { "input-with-icon": !!iconSvg },
      { [`${baseClass}__input--error`]: error },
      { [`${baseClass}__input--password`]: type === "password" && value }
    );

    const inputWrapperClasses = classnames(`${baseClass}__input-wrapper`, {
      [`${baseClass}__input-wrapper--disabled`]: disabled,
    });

    const iconClasses = classnames(
      `${baseClass}__icon`,
      { [`${baseClass}__icon--error`]: error },
      { [`${baseClass}__icon--active`]: value }
    );

    const handleClear = () => {
      onChange("");
    };

    return (
      <div className={wrapperClasses}>
        {this.props.label && this.renderHeading()}
        <div className={inputWrapperClasses}>
          <input
            id={name}
            name={name}
            onChange={onInputChange}
            onClick={onClick}
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
            data-1p-ignore={ignore1Password}
          />
          {iconSvg && <Icon name={iconSvg} className={iconClasses} />}
          {clearButton && !!value && (
            <Button
              onClick={() => handleClear()}
              variant="icon"
              className={`${baseClass}__clear-button`}
            >
              <Icon name="close-filled" color="core-fleet-black" />
            </Button>
          )}
        </div>
        {renderHelpText()}
      </div>
    );
  }
}

export default InputFieldWithIcon;
