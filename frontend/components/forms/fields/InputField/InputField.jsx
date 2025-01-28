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
    /** readOnly displays a non-editable field */
    readOnly: PropTypes.bool,
    /** disabled displays a greyed out non-editable field */
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
    tooltip: PropTypes.oneOfType([PropTypes.string, PropTypes.object]),
    labelTooltipPosition: PropTypes.string,
    helpText: PropTypes.oneOfType([
      PropTypes.string,
      PropTypes.arrayOf(PropTypes.string),
      PropTypes.object,
    ]),
    enableCopy: PropTypes.bool,
    copyButtonPosition: PropTypes.oneOf(["inside", "outside"]),
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
    labelTooltipPosition: undefined,
    helpText: "",
    enableCopy: false,
    copyButtonPosition: "outside",
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

  renderCopyButton = () => {
    const { value, copyButtonPosition } = this.props;

    const copyValue = (e) => {
      e.preventDefault();
      stringToClipboard(value).then(() => {
        this.setState({ copied: true });
        setTimeout(() => {
          this.setState({ copied: false });
        }, 2000);
      });
    };

    const copyButtonValue =
      copyButtonPosition === "outside" ? (
        <>
          <Icon name="copy" />
          <span>Copy</span>
        </>
      ) : (
        <Icon name="copy" />
      );

    const wrapperClasses = classnames(
      `${baseClass}__copy-wrapper`,
      copyButtonPosition === "outside"
        ? `${baseClass}__copy-wrapper-outside`
        : `${baseClass}__copy-wrapper-inside`
    );

    const copiedConfirmationClasses = classnames(
      `${baseClass}__copied-confirmation`,
      copyButtonPosition === "outside"
        ? `${baseClass}__copied-confirmation-outside`
        : `${baseClass}__copied-confirmation-inside`
    );

    return (
      <div className={wrapperClasses}>
        <Button
          variant="text-icon"
          onClick={copyValue}
          className={`${baseClass}__copy-value-button`}
        >
          {copyButtonValue}
        </Button>
        {this.state.copied && (
          <span className={copiedConfirmationClasses}>Copied!</span>
        )}
      </div>
    );
  };

  render() {
    const {
      readOnly,
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
      enableCopy,
      copyButtonPosition,
    } = this.props;

    const { onInputChange } = this;
    const shouldShowPasswordClass = type === "password";
    const inputClasses = classnames(baseClass, inputClassName, {
      [`${baseClass}--password`]: shouldShowPasswordClass,
      [`${baseClass}--read-only`]: readOnly || disabled,
      [`${baseClass}--disabled`]: disabled,
      [`${baseClass}--error`]: error,
      [`${baseClass}__textarea`]: type === "textarea",
    });

    const inputWrapperClasses = classnames(inputWrapperClass, {
      [`input-field--read-only`]: readOnly || disabled,
      [`input-field--disabled`]: disabled,
    });

    const formFieldProps = pick(this.props, [
      "helpText",
      "label",
      "error",
      "name",
      "tooltip",
      "labelTooltipPosition",
    ]);

    // FIXME: Why doesn't this pass onBlur and other props down if the type is textarea. Do we want
    // to change that? What might break if we do?

    if (type === "textarea") {
      return (
        <FormField
          {...formFieldProps}
          type="textarea"
          className={inputWrapperClasses}
        >
          <textarea
            name={name}
            id={name}
            onChange={onInputChange}
            className={inputClasses}
            disabled={readOnly || disabled}
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
      "copy-enabled": enableCopy,
      "copy-outside": enableCopy && copyButtonPosition === "outside",
      "copy-inside": enableCopy && copyButtonPosition === "inside",
    });

    return (
      <FormField
        {...formFieldProps}
        type="input"
        className={inputWrapperClasses}
      >
        <div className={inputContainerClasses}>
          <input
            disabled={readOnly || disabled}
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

          {enableCopy && this.renderCopyButton()}
        </div>
      </FormField>
    );
  }
}

export default InputField;
