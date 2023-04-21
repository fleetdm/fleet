import React from "react";

import { BootstrapPackageStatus } from "interfaces/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

const baseClass = "bootstrap-package-indicator";

const STATUS_DISPLAY_OPTIONS = {
  installed: {
    iconName: "success",
    displayText: "Installed",
  },
  pending: {
    iconName: "pending",
    displayText: "Pending",
  },
  failed: {
    iconName: "error",
    displayText: "Failed",
  },
} as const;

interface IBootstrapPackageIndicatorProps {
  status: BootstrapPackageStatus;
  details: string;
  onClick: (details: string) => void;
}

const BootstrapPackageIndicator = ({
  status,
  details,
  onClick,
}: IBootstrapPackageIndicatorProps) => {
  const displayData = STATUS_DISPLAY_OPTIONS[status];

  return (
    <div className={baseClass}>
      <Icon name={displayData.iconName} />
      <span>
        <Button
          onClick={onClick}
          variant="text-link"
          className={`${baseClass}__button`}
        >
          {displayData.displayText}
        </Button>
      </span>
    </div>
  );
};

export default BootstrapPackageIndicator;
