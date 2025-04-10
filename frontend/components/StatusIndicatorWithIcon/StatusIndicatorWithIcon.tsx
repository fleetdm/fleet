import React from "react";
import classnames from "classnames";

import { IconNames } from "components/icons";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "status-indicator-with-icon";

export type IndicatorStatus =
  | "success"
  | "successPartial"
  | "pending"
  | "pendingPartial"
  | "error";

interface IStatusIndicatorWithIconProps {
  status: IndicatorStatus;
  value: string;
  tooltip?: {
    tooltipText: string | JSX.Element;
    position?: "top" | "bottom";
  };
  layout?: "horizontal" | "vertical";
  className?: string;
  /** Classname to add to the value text */
  valueClassName?: string;
}

const statusIconNameMapping: Record<IndicatorStatus, IconNames> = {
  success: "success",
  successPartial: "success-outline",
  pending: "pending",
  pendingPartial: "pending-outline",
  error: "error",
};

const StatusIndicatorWithIcon = ({
  status,
  value,
  tooltip,
  layout = "horizontal",
  className,
  valueClassName,
}: IStatusIndicatorWithIconProps) => {
  const classNames = classnames(baseClass, className);

  const valueClasses = classnames(`${baseClass}__value`, valueClassName, {
    [`${baseClass}__value-vertical`]: layout === "vertical",
  });
  const valueContent = (
    <span className={valueClasses}>
      <Icon name={statusIconNameMapping[status]} />
      <span>{value}</span>
    </span>
  );

  const indicatorContent = tooltip ? (
    <TooltipWrapper
      className={`${baseClass}__tooltip`}
      tooltipClass="indicator-tip-text"
      position="top"
      tipContent={tooltip.tooltipText}
      tipOffset={10}
      showArrow
      underline={false}
      fixedPositionStrategy
    >
      {valueContent}
    </TooltipWrapper>
  ) : (
    <span>{valueContent}</span>
  );

  return <div className={classNames}>{indicatorContent}</div>;
};

export default StatusIndicatorWithIcon;
