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
  error?: string | null;
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
  value?: boolean | string | number | null;
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

  // Old-style icon copy button for textarea (positioned absolutely above textarea)
  const renderTextareaCopyButton = () => {
    return (
      <div
        className={`${baseClass}__copy-wrapper ${baseClass}__copy-wrapper--text-area`}
      >
        {copied && (
          <span className={`${baseClass}__copied-confirmation`}>Copied!</span>
        )}
        <Button variant="icon" onClick={onClickCopy} size="small" iconStroke>
          <Icon name="copy" />
        </Button>
      </div>
    );
  };

  // New bordered action buttons for input fields
  const renderActionButtons = () => {
    return (
      <div className={`${baseClass}__action-buttons`}>
        {enableCopy && (
          <div className={`${baseClass}__action-button-wrapper`}>
            {copied && (
              <span className={`${baseClass}__copied-confirmation`}>
                Copied!
              </span>
            )}
            <button
              type="button"
              className={`${baseClass}__action-button`}
              onClick={onClickCopy}
              aria-label="Copy to clipboard"
            >
              <Icon name="copy" />
            </button>
          </div>
        )}
        {enableShowSecret && (
          <button
            type="button"
            className={`${baseClass}__action-button`}
            onClick={onToggleSecret}
            aria-label={showSecret ? "Hide secret" : "Show secret"}
            aria-pressed={showSecret}
          >
            <Icon name="eye" />
          </button>
        )}
      </div>
    );
  };

  const shouldShowPasswordClass = type === "password" && !showSecret;
  const inputClasses = classnames(baseClass, inputClassName, {
    [`${baseClass}--password`]: shouldShowPasswordClass,
    [`${baseClass}--read-only`]: readOnly || disabled,
    [`${baseClass}--disabled`]: disabled,
    [`${baseClass}--error`]: !!error,
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

  const hasActionButtons = enableCopy || enableShowSecret;

  if (type === "textarea") {
    const textareaContainerClasses = classnames(
      `${baseClass}__input-container`,
      { "copy-enabled": enableCopy }
    );

    return (
      <FormField
        {...formFieldProps}
        type="textarea"
        className={inputWrapperClasses}
      >
        <div className={textareaContainerClasses}>
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
          {enableCopy && renderTextareaCopyButton()}
        </div>
      </FormField>
    );
  }

  const inputType = showSecret ? "text" : type;
  const inputContainerClasses = classnames(`${baseClass}__input-container`, {
    [`${baseClass}__input-container--has-actions`]: hasActionButtons,
  });

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
        {hasActionButtons && renderActionButtons()}
      </div>
    </FormField>
  );
};

export default InputField;
