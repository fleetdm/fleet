import React from "react";

import { BootstrapPackageStatus } from "interfaces/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "bootstrap-package-indicator";

const STATUS_DISPLAY_OPTIONS = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tipContent: (
      <span className={`${baseClass}__tooltip`}>
        The host acknowledged the MDM command to install bootstrap package.
      </span>
    ),
  },
  pending: {
    iconName: "pending",
    displayText: "Pending",
    tipContent: (
      <span className={`${baseClass}__tooltip`}>
        Bootstrap package is installing or will install when the host comes
        online.
      </span>
    ),
  },
  failed: {
    iconName: "error",
    displayText: "Failed",
    tipContent: (
      <span className={`${baseClass}__tooltip`}>
        The host failed to install bootstrap package. To view errors, select{" "}
        <b>Failed</b>.
      </span>
    ),
  },
} as const;

interface IBootstrapPackageIndicatorProps {
  status: BootstrapPackageStatus;
  onClick?: () => void;
}

const BootstrapPackageIndicator = ({
  status,
  onClick,
}: IBootstrapPackageIndicatorProps) => {
  const displayData = STATUS_DISPLAY_OPTIONS[status];

  return (
    <div className={baseClass}>
      <Icon name={displayData.iconName} />
      <span>
        <TooltipWrapper
          position="top"
          showArrow
          tipContent={displayData.tipContent}
          underline={false}
        >
          {status !== BootstrapPackageStatus.FAILED ? (
            <>{displayData.displayText}</>
          ) : (
            <Button
              onClick={onClick}
              variant="text-link"
              className={`${baseClass}__button`}
            >
              {displayData.displayText}
            </Button>
          )}
        </TooltipWrapper>
      </span>
    </div>
  );
};

export default BootstrapPackageIndicator;
