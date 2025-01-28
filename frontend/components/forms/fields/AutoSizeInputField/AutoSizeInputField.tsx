import React, { KeyboardEvent, useEffect, useRef } from "react";
import classnames from "classnames";

interface IAutoSizeInputFieldProps {
  name: string;
  placeholder: string;
  value: string;
  inputClassName?: string;
  maxLength: number;
  hasError?: boolean;
  isDisabled?: boolean;
  isFocused?: boolean;
  onFocus?: () => void;
  onBlur?: () => void;
  onChange: (newSelectedValue: string) => void;
  onKeyPress: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
}

const baseClass = "component__auto-size-input-field";

const AutoSizeInputField = ({
  name,
  placeholder,
  value,
  inputClassName,
  maxLength,
  hasError,
  isDisabled,
  isFocused,
  onFocus = () => null,
  onBlur = () => null,
  onChange,
  onKeyPress,
}: IAutoSizeInputFieldProps): JSX.Element => {
  const inputClasses = classnames(baseClass, inputClassName, "no-hover", {
    [`${baseClass}--disabled`]: isDisabled,
    [`${baseClass}--error`]: hasError,
    [`${baseClass}__textarea`]: true,
  });

  const inputElement = useRef<any>(null);

  useEffect(() => {
    if (isFocused && inputElement.current) {
      inputElement.current.focus();
      inputElement.current.selectionStart = value.length;
      inputElement.current.selectionEnd = value.length;
    }
  }, [isFocused]);

  const onInputFocus = () => {
    isFocused = true;
    onFocus();
  };

  const onInputBlur = () => {
    isFocused = false;
    onBlur();
  };

  const onInputKeyPress = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    onKeyPress(event);
  };

  const onInputChange = (event: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange(event.target.value);
  };

  return (
    <div className={baseClass}>
      <label className="input-sizer" data-value={value} htmlFor={name}>
        <textarea
          name={name}
          id={name}
          onChange={onInputChange}
          placeholder={placeholder}
          value={value}
          maxLength={maxLength}
          className={inputClasses}
          cols={1}
          rows={1}
          tabIndex={0}
          onFocus={onInputFocus}
          onBlur={onInputBlur}
          onKeyPress={onInputKeyPress}
          ref={inputElement}
        />
      </label>
    </div>
  );
};

export default AutoSizeInputField;
