import React from "react";

import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "managed-account-checkbox";

interface IManagedAccountCheckboxProps {
  value: boolean;
  onChange: (value: boolean) => void;
  disabled: boolean;
  /** Rendered on the checkbox icon, e.g. to explain why the box cannot be unchecked. */
  iconTooltipContent?: React.ReactNode;
}

const ManagedAccountCheckbox = ({
  value,
  onChange,
  disabled,
  iconTooltipContent,
}: IManagedAccountCheckboxProps) => {
  return (
    <Checkbox
      className={baseClass}
      disabled={disabled}
      iconTooltipContent={iconTooltipContent}
      value={value}
      onChange={onChange}
      helpText="A hidden local admin for remote troubleshooting."
      labelTooltipContent={
        <>
          Fleet creates a user (_fleetadmin) and unique password for each host,
          accessible in <b>Host details &gt; Show managed account</b>.
        </>
      }
    >
      Create hidden admin
    </Checkbox>
  );
};

export default ManagedAccountCheckbox;
