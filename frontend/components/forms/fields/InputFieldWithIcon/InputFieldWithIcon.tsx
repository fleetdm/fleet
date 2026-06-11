import React, { useCallback, useEffect, useRef } from "react";
import classnames from "classnames";

import { ICON_MAP } from "components/icons";
import Icon from "components/Icon/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";

const baseClass = "input-icon-field";

type IconName = keyof typeof ICON_MAP;

export interface IInputFieldWithIconProps {
  autofocus?: boolean;
  error?: string | null;
  helpText?: string[] | string;
  iconSvg?: IconName;
  label?: string;
  name?: string;
  onChange?: (value: string) => void;
  onClick?: (evt: React.MouseEvent<HTMLInputElement>) => void;
  clearButton?: boolean;
  placeholder?: string;
  tabIndex?: number;
  type?: string;
  className?: string;
  disabled?: boolean;
  inputOptions?: React.InputHTMLAttributes<HTMLInputElement>;
  tooltip?: string;
  /**
   * Whether 1Password should skip this field.
   * Defaults to `true` because most Fleet inputs are not credential fields.
   */
  ignore1Password?: boolean;
  value?: string;
}

const InputFieldWithIcon = ({
  autofocus = false,
  error,
  helpText,
  iconSvg,
  label,
  name,
  onChange,
  onClick,
  clearButton,
  placeholder,
  tabIndex,
  type,
  className,
  disabled,
  inputOptions,
  tooltip,
  ignore1Password = true,
  value,
}: IInputFieldWithIconProps): JSX.Element => {
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (autofocus && inputRef.current) {
      inputRef.current.focus();
    }
  }, [autofocus]);

  const onInputChange = useCallback(
    (evt: React.ChangeEvent<HTMLInputElement>) => {
      evt.preventDefault();

      return onChange?.(evt.target.value);
    },
    [onChange]
  );

  const renderHeading = () => {
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

  const renderHelpText = () => {
    if (helpText) {
      return (
        <span className={`${baseClass}__help-text form-field__help-text`}>
          {helpText}
        </span>
      );
    }

    return false;
  };

  const wrapperClasses = classnames(baseClass, "form-field");

  const inputClasses = classnames(
    `${baseClass}__input`,
    className,
    { "input-with-icon": !!iconSvg },
    { [`${baseClass}__input--error`]: !!error },
    { [`${baseClass}__input--password`]: !!(type === "password" && value) }
  );

  const inputWrapperClasses = classnames(`${baseClass}__input-wrapper`, {
    [`${baseClass}__input-wrapper--disabled`]: disabled,
  });

  const iconClasses = classnames(
    `${baseClass}__icon`,
    { [`${baseClass}__icon--error`]: !!error },
    { [`${baseClass}__icon--active`]: !!value }
  );

  const handleClear = () => {
    onChange?.("");
  };

  return (
    <div className={wrapperClasses}>
      {label && renderHeading()}
      <div className={inputWrapperClasses}>
        <input
          id={name}
          name={name}
          onChange={onInputChange}
          onClick={onClick}
          className={inputClasses}
          placeholder={placeholder}
          ref={inputRef}
          tabIndex={tabIndex}
          type={type}
          value={value}
          disabled={disabled}
          {...inputOptions}
          // 1Password only checks for the presence (not the value) of `data-1p-ignore`,
          // so we omit the attribute entirely when the field is not meant to be ignored.
          // See https://developer.1password.com/docs/web/compatible-website-design/
          data-1p-ignore={ignore1Password || undefined}
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
};

export default InputFieldWithIcon;
