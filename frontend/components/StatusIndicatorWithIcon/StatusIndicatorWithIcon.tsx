import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";
import classnames from "classnames";

import { IconNames } from "components/icons";
import Icon from "components/Icon";
import { COLORS } from "styles/var/colors";

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
  const id = `status-${uniqueId()}`;

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
    <>
      <span data-tip data-for={id}>
        {valueContent}
      </span>
      <ReactTooltip
        className={`${baseClass}__tooltip`}
        place={tooltip?.position ? tooltip.position : "top"}
        type="dark"
        effect="solid"
        id={id}
        backgroundColor={COLORS["tooltip-bg"]}
      >
        {tooltip.tooltipText}
      </ReactTooltip>
    </>
  ) : (
    <span>{valueContent}</span>
  );

  // FIXME: It seems like this needs to include the __value class to work properly (otherwise the
  // icon formatting is off).
  return <div className={classNames}>{indicatorContent}</div>;
};

export default StatusIndicatorWithIcon;
