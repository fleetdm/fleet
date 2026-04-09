import React, { useState, useRef, useEffect, useCallback } from "react";
import classnames from "classnames";

import { PlacesType } from "react-tooltip-5";

import { stringToClipboard } from "utilities/copy_text";

import FormField from "components/forms/FormField";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "input-field";

export interface IInputFieldProps {
  autofocus?: boolean;
  /** readOnly displays a non-editable field */
  readOnly?: boolean;
  /** disabled displays a greyed out non-editable field */
  disabled?: boolean;
  error?: string;
  inputClassName?: string;
  inputWrapperClass?: string;
  inputOptions?: React.InputHTMLAttributes<HTMLInputElement>;
  name?: string;
  /**
   * Receives the field value (string) by default, or { name, value } when
   * parseTarget is true. See IInputFieldParseTarget and InputFieldOnChange
   * in interfaces/form_field.ts for caller-side typing helpers.
   */
  onChange?: (value: any) => void;
  onBlur?: (
    evt: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => void;
  onFocus?: (
    evt: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => void;
  placeholder?: string;
  type?: string;
  blockAutoComplete?: boolean;
  value: boolean | string | number;
  /** Returns both name and value */
  parseTarget?: boolean;
  tooltip?: React.ReactNode;
  labelTooltipPosition?: PlacesType;
  label?: React.ReactNode;
  labelClassName?: string;
  helpText?: React.ReactNode;
  /** Use in conjunction with type "password" and enableCopy to see eye icon to view */
  enableShowSecret?: boolean;
  enableCopy?: boolean;
  ignore1password?: boolean;
  /** Only effective on input type number */
  step?: string | number;
  /** Only effective on input type number */
  min?: string | number;
  /** Only effective on input type number */
  max?: string | number;
}

const InputField = ({
  autofocus = false,
  readOnly,
  disabled,
  error,
  inputClassName,
  inputWrapperClass = "",
  inputOptions = {},
  name,
  onChange,
  onBlur,
  onFocus,
  placeholder,
  type = "text",
  blockAutoComplete = false,
  value = "",
  parseTarget = false,
  tooltip = "",
  labelTooltipPosition,
  label = null,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  labelClassName: _labelClassName = "",
  helpText = "",
  enableShowSecret = false,
  enableCopy = false,
  ignore1password = false,
  step,
  min,
  max,
}: IInputFieldProps): JSX.Element => {
  const [copied, setCopied] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null);

  useEffect(() => {
    if (autofocus && inputRef.current) {
      (inputRef.current as HTMLElement).focus();
    }
  }, [autofocus]);

  const onInputChange = useCallback(
    (evt: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      evt.preventDefault();

      const target = evt.target as HTMLInputElement;
      const { value: inputValue, name: inputName } = target;

      if (parseTarget) {
        // Returns both name and value
        return onChange?.({ value: inputValue, name: inputName });
      }

      return onChange?.(inputValue);
    },
    [onChange, parseTarget]
  );

  const onToggleSecret = useCallback((evt: React.MouseEvent) => {
    evt.preventDefault();
    setShowSecret((prev) => !prev);
  }, []);

  const onClickCopy = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      stringToClipboard(value).then(() => {
        setCopied(true);
        setTimeout(() => {
          setCopied(false);
        }, 2000);
      });
    },
    [value]
  );

  const renderShowSecretButton = () => {
    return (
      <Button
        variant="icon"
        className={`${baseClass}__show-secret-icon`}
        onClick={onToggleSecret}
        size="small"
      >
        <Icon name="eye" />
      </Button>
    );
  };

  const renderCopyButton = () => {
    const copyButtonValue = <Icon name="copy" />;
    const wrapperClasses = classnames(`${baseClass}__copy-wrapper`, {
      [`${baseClass}__copy-wrapper__text-area`]: type === "textarea",
    });

    const copiedConfirmationClasses = classnames(
      `${baseClass}__copied-confirmation`
    );

    return (
      <div className={wrapperClasses}>
        {copied && <span className={copiedConfirmationClasses}>Copied!</span>}
        <Button variant="icon" onClick={onClickCopy} size="small" iconStroke>
          {copyButtonValue}
        </Button>
        {enableShowSecret && renderShowSecretButton()}
      </div>
    );
  };

  const shouldShowPasswordClass = type === "password" && !showSecret;
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

  const formFieldProps = {
    helpText,
    label,
    error,
    name: name ?? "",
    tooltip,
    labelTooltipPosition,
  };

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
            onBlur={onBlur}
            onFocus={onFocus}
            className={inputClasses}
            disabled={readOnly || disabled}
            placeholder={placeholder}
            ref={(r) => {
              inputRef.current = r;
            }}
            {...(inputOptions as React.TextareaHTMLAttributes<HTMLTextAreaElement>)}
            value={value as string | number}
          />
          {enableCopy && renderCopyButton()}
        </div>
      </FormField>
    );
  }

  const inputType = showSecret ? "text" : type;

  return (
    <FormField {...formFieldProps} type="input" className={inputWrapperClasses}>
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
            inputRef.current = r;
          }}
          type={inputType}
          {...inputOptions}
          value={value as string | number}
          autoComplete={blockAutoComplete ? "new-password" : ""}
          data-1p-ignore={ignore1password}
          step={step}
          min={min}
          max={max}
        />
        {enableCopy && renderCopyButton()}
      </div>
    </FormField>
  );
};

export default InputField;
