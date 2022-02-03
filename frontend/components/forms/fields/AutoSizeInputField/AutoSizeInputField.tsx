import React, {
  ChangeEvent,
  KeyboardEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import classnames from "classnames";

interface IAutoSizeInputFieldProps {
  name: string;
  placeholder: string;
  value: string;
  inputClassName?: string;
  hasError?: boolean;
  isDisabled?: boolean;
  isFocused?: boolean;
  onFocus: () => void;
  onBlur: () => void;
  onChange: (newSelectedValue: string) => void;
  onKeyPress: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
}

const baseClass = "component__auto-size-input-field";

const TeamsDropdown = ({
  name,
  placeholder,
  value,
  inputClassName,
  hasError,
  isDisabled,
  onFocus,
  onBlur,
  onChange,
  onKeyPress,
  isFocused,
}: IAutoSizeInputFieldProps): JSX.Element => {
  const [inputValue, setInputValue] = useState(value);

  const inputClasses = classnames(baseClass, inputClassName, "no-hover", {
    [`${baseClass}--disabled`]: isDisabled,
    [`${baseClass}--error`]: hasError,
    [`${baseClass}__textarea`]: true,
  });

  const inputElement = useRef<any>(null);

  useEffect(() => {
    onChange(inputValue);
  }, [inputValue]);

  useEffect(() => {
    if (isFocused && inputElement.current) {
      inputElement.current.focus();
      inputElement.current.selectionStart = inputValue.length;
      inputElement.current.selectionEnd = inputValue.length;
    }
  }, [isFocused]);

  const onInputChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setInputValue(event.currentTarget.value);
  };

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

  return (
    <div className={baseClass}>
      <label className="input-sizer" data-value={inputValue} htmlFor={name}>
        <textarea
          name={name}
          id={name}
          onChange={onInputChange}
          placeholder={placeholder}
          value={inputValue}
          className={inputClasses}
          cols={12}
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

export default TeamsDropdown;
