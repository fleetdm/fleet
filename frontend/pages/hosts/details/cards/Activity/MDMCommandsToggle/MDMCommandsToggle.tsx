import React from "react";
import classnames from "classnames";

import Slider from "components/forms/fields/Slider";

const baseClass = "mdm-commands-toggle";

interface IMDMCommandsToggleProps {
  showMDMCommands: boolean;
  commandCount?: number;
  className?: string;
  onToggleMDMCommands: () => void;
}

const MDMCommandsToggle = ({
  showMDMCommands,
  commandCount,
  className,
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
      onChange={onToggleMDMCommands}
    />
  );
};

export default MDMCommandsToggle;
