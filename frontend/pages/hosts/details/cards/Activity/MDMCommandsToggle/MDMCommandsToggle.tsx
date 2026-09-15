import classnames from "classnames";
import React from "react";

import Slider from "components/forms/fields/Slider";

const baseClass = "mdm-commands-toggle";

interface IMDMCommandsToggleProps {
  showMDMCommands: boolean;
  commandCount?: number;
  className?: string;
  disabled?: boolean;
  /** Shown on hover over the label, and readable while disabled. */
  labelTooltip?: JSX.Element | string;
  onToggleMDMCommands: () => void;
}

const MDMCommandsToggle = ({
  showMDMCommands,
  commandCount,
  className,
  disabled,
  labelTooltip,
  onToggleMDMCommands,
}: IMDMCommandsToggleProps) => {
  const classNames = classnames(baseClass, className);
  const labelText = `Show MDM commands${
    commandCount !== undefined ? ` (${commandCount})` : ""
  }`;

  return (
    <Slider
      className={classNames}
      activeText={labelText}
      inactiveText={labelText}
      value={showMDMCommands}
      disabled={disabled}
      labelTooltip={labelTooltip}
      onChange={onToggleMDMCommands}
    />
  );
};

export default MDMCommandsToggle;
