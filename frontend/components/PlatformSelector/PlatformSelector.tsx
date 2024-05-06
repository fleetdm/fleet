import React from "react";
import classNames from "classnames";
import Checkbox from "components/forms/fields/Checkbox";

interface IPlatformSelectorProps {
  baseClass?: string;
  checkDarwin: boolean;
  checkWindows: boolean;
  checkLinux: boolean;
  checkChrome: boolean;
  setCheckDarwin: (val: boolean) => void;
  setCheckWindows: (val: boolean) => void;
  setCheckLinux: (val: boolean) => void;
  setCheckChrome: (val: boolean) => void;
  disabled?: boolean;
}

export const PlatformSelector = ({
  baseClass: parentClass,
  checkDarwin,
  checkWindows,
  checkLinux,
  checkChrome,
  setCheckDarwin,
  setCheckWindows,
  setCheckLinux,
  setCheckChrome,
  disabled = false,
}: IPlatformSelectorProps): JSX.Element => {
  const baseClass = "platform-selector";

  const labelClasses = classNames("form-field__label", {
    [`form-field__label--disabled`]: disabled,
  });

  return (
    <div className={`${parentClass}__${baseClass} ${baseClass} form-field`}>
      <span className={labelClasses}>Checks on:</span>
      <span className={`${baseClass}__checkboxes`}>
        <Checkbox
          value={checkDarwin}
          onChange={(value: boolean) => setCheckDarwin(value)}
          wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          disabled={disabled}
        >
          macOS
        </Checkbox>
        <Checkbox
          value={checkWindows}
          onChange={(value: boolean) => setCheckWindows(value)}
          wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          disabled={disabled}
        >
          Windows
        </Checkbox>
        <Checkbox
          value={checkLinux}
          onChange={(value: boolean) => setCheckLinux(value)}
          wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          disabled={disabled}
        >
          Linux
        </Checkbox>
        <Checkbox
          value={checkChrome}
          onChange={(value: boolean) => setCheckChrome(value)}
          wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          disabled={disabled}
        >
          ChromeOS
        </Checkbox>
      </span>
      <div className="form-field__help-text">
        Your policy will only be checked on the selected platform(s).
      </div>
    </div>
  );
};

export default PlatformSelector;
