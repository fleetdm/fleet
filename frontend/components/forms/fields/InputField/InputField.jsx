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
    /** Use in conjunction with type "password" and enableCopy to see eye icon to view */
    enableShowSecret: PropTypes.bool,
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
    labelTooltipPosition: undefined,
    helpText: "",
    enableCopy: false,
    enableShowSecret: false,
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

  onToggleSecret = (evt) => {
    evt.preventDefault();

    this.setState({ showSecret: !this.state.showSecret });
    return false;
  };

  onClickCopy = (e) => {
    e.preventDefault();
    stringToClipboard(this.props.value).then(() => {
      this.setState({ copied: true });
      setTimeout(() => {
        this.setState({ copied: false });
      }, 2000);
    });
  };

  renderShowSecretButton = () => {
    const { onToggleSecret } = this;

    return (
      <Button
        variant="icon"
        className={`${baseClass}__show-secret-icon`}
        onClick={onToggleSecret}
      >
        <Icon name="eye" />
      </Button>
    );
  };

  renderCopyButton = () => {
    const { onClickCopy } = this;

    const copyButtonValue = <Icon name="copy" />;
    const wrapperClasses = classnames(`${baseClass}__copy-wrapper`);

    const copiedConfirmationClasses = classnames(
      `${baseClass}__copied-confirmation`
    );

    return (
      <div className={wrapperClasses}>
        {this.state.copied && (
          <span className={copiedConfirmationClasses}>Copied!</span>
        )}
        <Button variant={"icon"} onClick={onClickCopy} iconStroke>
          {copyButtonValue}
        </Button>
        {this.props.enableShowSecret && this.renderShowSecretButton()}
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
      enableShowSecret,
    } = this.props;

    const { onInputChange } = this;
    const shouldShowPasswordClass =
      type === "password" && !this.state.showSecret;
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

    const inputContainerClasses = classnames(`${baseClass}__input-container`, {
      "copy-enabled": enableCopy,
    });

    if (type === "textarea") {
      return (
        <FormField
          {...formFieldProps}
          type="textarea"
          className={inputWrapperClasses}
        >
          <div className={inputContainerClasses}>
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
            {enableCopy && this.renderCopyButton()}
          </div>
        </FormField>
      );
    }

    const inputType = this.state.showSecret ? "text" : type;

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
            type={inputType}
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
