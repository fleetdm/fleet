import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { noop, pick } from "lodash";

import { stringToClipboard } from "utilities/copy_text";

import FormField from "components/forms/FormField";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "input-field";

class InputField extends Component {
  static propTypes = {
    autofocus: PropTypes.bool,
    disabled: PropTypes.bool,
    error: PropTypes.string,
    inputClassName: PropTypes.string, // eslint-disable-line react/forbid-prop-types
    inputWrapperClass: PropTypes.string,
    inputOptions: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    name: PropTypes.string,
    onChange: PropTypes.func,
    onBlur: PropTypes.func,
    onFocus: PropTypes.func,
    placeholder: PropTypes.string,
    type: PropTypes.string,
    blockAutoComplete: PropTypes.bool,
    value: PropTypes.oneOfType([
      PropTypes.bool,
      PropTypes.string,
      PropTypes.number,
    ]).isRequired,
    parseTarget: PropTypes.bool,
    tooltip: PropTypes.string,
    labelTooltipPosition: PropTypes.string,
    helpText: PropTypes.oneOfType([
      PropTypes.string,
      PropTypes.arrayOf(PropTypes.string),
      PropTypes.object,
    ]),
    enableCopy: PropTypes.bool,
    ignore1password: PropTypes.bool,
  };

  static defaultProps = {
    autofocus: false,
    inputWrapperClass: "",
    inputOptions: {},
    label: null,
    labelClassName: "",
    onFocus: noop,
    onBlur: noop,
    type: "text",
    blockAutoComplete: false,
    value: "",
    parseTarget: false,
    tooltip: "",
    labelTooltipPosition: "",
    helpText: "",
    enableCopy: false,
    ignore1password: false,
  };

  constructor() {
    super();
    this.state = {
      copied: false,
    };
  }

  componentDidMount() {
    const { autofocus } = this.props;
    const { input } = this;

    if (autofocus) {
      input.focus();
    }

    return false;
  }

  onInputChange = (evt) => {
    evt.preventDefault();

    const { value, name } = evt.target;
    const { onChange, parseTarget } = this.props;

    if (parseTarget) {
      // Returns both name and value
      return onChange({ value, name });
    }

    return onChange(value);
  };

  render() {
    const {
      disabled,
      error,
      inputClassName,
      inputOptions,
      inputWrapperClass,
      name,
      onFocus,
      onBlur,
      placeholder,
      type,
      blockAutoComplete,
      value,
      ignore1password,
    } = this.props;

    const { onInputChange } = this;
    const shouldShowPasswordClass = type === "password";
    const inputClasses = classnames(baseClass, inputClassName, {
      [`${baseClass}--password`]: shouldShowPasswordClass,
      [`${baseClass}--disabled`]: disabled,
      [`${baseClass}--error`]: error,
      [`${baseClass}__textarea`]: type === "textarea",
    });

    const formFieldProps = pick(this.props, [
      "helpText",
      "label",
      "error",
      "name",
      "tooltip",
      "labelTooltipPosition",
    ]);

    const copyValue = (e) => {
      e.preventDefault();
      stringToClipboard(value).then(() => {
        this.setState({ copied: true });
        setTimeout(() => {
          this.setState({ copied: false });
        }, 2000);
      });
    };

    if (type === "textarea") {
      return (
        <FormField
          {...formFieldProps}
          type="textarea"
          className={inputWrapperClass}
        >
          <textarea
            name={name}
            id={name}
            onChange={onInputChange}
            className={inputClasses}
            disabled={disabled}
            placeholder={placeholder}
            ref={(r) => {
              this.input = r;
            }}
            type={type}
            {...inputOptions}
            value={value}
          />
        </FormField>
      );
    }

    const inputContainerClasses = classnames(`${baseClass}__input-container`, {
      "copy-enabled": this.props.enableCopy,
    });

    return (
      <FormField {...formFieldProps} type="input" className={inputWrapperClass}>
        <div className={inputContainerClasses}>
          <input
            disabled={disabled}
            name={name}
            id={name}
            onChange={onInputChange}
            onFocus={onFocus}
            onBlur={onBlur}
            className={inputClasses}
            placeholder={placeholder}
            ref={(r) => {
              this.input = r;
            }}
            type={type}
            {...inputOptions}
            value={value}
            autoComplete={blockAutoComplete ? "new-password" : ""}
            data-1p-ignore={ignore1password}
          />
          {this.props.enableCopy && (
            <div className={`${baseClass}__copy-wrapper`}>
              <Button
                variant="text-icon"
                onClick={copyValue}
                className={`${baseClass}__copy-value-button`}
              >
                <Icon name="copy" /> Copy
              </Button>
              {this.state.copied && (
                <span className={`${baseClass}__copied-confirmation`}>
                  Copied!
                </span>
              )}
            </div>
          )}
        </div>
      </FormField>
    );
  }
}

export default InputField;
