import React, { ReactNode } from "react";
import classnames from "classnames";

import StatusIndicator from "components/StatusIndicator";

import { monthDayYearFormat } from "utilities/date_format";
import { hasLicenseExpired, willExpireWithinXDays } from "utilities/helpers";
import { IIndicatorValue } from "components/StatusIndicator/StatusIndicator";

const baseClass = "renew-date-cell";

export type IRenewDateCellStatusConfig = Record<
  Exclude<IIndicatorValue, "indeterminate" | "success">,
  {
    tooltipText: ReactNode;
  }
>;

interface IRenewDateCellProps {
  value: string;
  /**
   * `statusConfig` currently this allows us to dynamically change the tooltip
   * text depending on the status of the date. Can be extended later if needed.
   */
  statusConfig: IRenewDateCellStatusConfig;
  className?: string;
}

const RenewDateCell = ({
  value,
  statusConfig,
  className,
}: IRenewDateCellProps) => {
  const formattedDate = monthDayYearFormat(value);

  // "w250" is a utility class that sets the width the the same as
  // the other text cells in the table.
  // TODO: consider creating a generic StatusIndicatorCell component that
  // take in a value, a desired status to display and, can contain these
  // table cell styles.
  const classNames = classnames(baseClass, className, "w250");

  let indicatorStatus: Exclude<IIndicatorValue, "indeterminate"> = "success";
  let tooltipText: ReactNode = null;

  if (willExpireWithinXDays(value, 30)) {
    indicatorStatus = "warning";
  } else if (hasLicenseExpired(value)) {
    indicatorStatus = "error";
  }

  if (indicatorStatus !== "success") {
    tooltipText = statusConfig[indicatorStatus].tooltipText;
  }

  const tooltipProp = tooltipText ? { tooltipText } : undefined;

  return (
    <StatusIndicator
      className={classNames}
      value={formattedDate}
      indicator={indicatorStatus}
      tooltip={tooltipProp}
    />
  );
};

export default RenewDateCell;
