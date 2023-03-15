import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";
import classnames from "classnames";

import { IconNames } from "components/icons";
import Icon from "components/Icon";
import { COLORS } from "styles/var/colors";

const baseClass = "status-indicator-with-icon";

type Status = "success" | "pending" | "error";

interface IStatusIndicatorWithIconProps {
  status: Status;
  value: string;
  tooltip?: {
    tooltipText: string | JSX.Element;
    position?: "top" | "bottom";
  };
  className?: string;
}

const statusIconNameMapping: Record<Status, IconNames> = {
  success: "success",
  pending: "pending",
  error: "error",
};

const StatusIndicatorWithIcon = ({
  status,
  value,
  tooltip,
  className,
}: IStatusIndicatorWithIconProps) => {
  const classNames = classnames(baseClass, className);
  const id = `status-${uniqueId()}`;

  const valueContent = (
    <span className={`${baseClass}__value`}>
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

  return <div className={classNames}>{indicatorContent}</div>;
};

export default StatusIndicatorWithIcon;
