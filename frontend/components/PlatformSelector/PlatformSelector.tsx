import React from "react";
import Checkbox from "components/forms/fields/Checkbox";

interface IPlatformSelectorProps {
  baseClass?: string;
  checkDarwin: boolean;
  checkWindows: boolean;
  checkLinux: boolean;
  setCheckDarwin: (val: boolean) => void;
  setCheckWindows: (val: boolean) => void;
  setCheckLinux: (val: boolean) => void;
}

export const PlatformSelector = ({
  baseClass: parentClass,
  checkDarwin,
  checkWindows,
  checkLinux,
  setCheckDarwin,
  setCheckWindows,
  setCheckLinux,
}: IPlatformSelectorProps): JSX.Element => {
  const baseClass = "platform-selector";

  return (
    <div className={`${parentClass}__${baseClass} ${baseClass}`}>
      <span>
        <b>Checks on:</b>
        <span className={`${baseClass}__checkboxes`}>
          <Checkbox
            value={checkDarwin}
            onChange={(value: boolean) => setCheckDarwin(value)}
            wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          >
            macOS
          </Checkbox>
          <Checkbox
            value={checkWindows}
            onChange={(value: boolean) => setCheckWindows(value)}
            wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          >
            Windows
          </Checkbox>
          <Checkbox
            value={checkLinux}
            onChange={(value: boolean) => setCheckLinux(value)}
            wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          >
            Linux
          </Checkbox>
        </span>
      </span>
      <p>Your policy will only be checked on the selected platform(s).</p>
    </div>
  );
};

export default PlatformSelector;
