import React from "react";
import classnames from "classnames";

import StatusIndicator from "components/StatusIndicator";

import { monthDayYearFormat } from "utilities/date_format";
import { hasLicenseExpired, willExpireWithinXDays } from "utilities/helpers";
import { IIndicatorValue } from "components/StatusIndicator/StatusIndicator";

const baseClass = "renew-date-cell";

interface IRenewDateCellProps {
  value: string;
  className?: string;
}

const RenewDateCell = ({ value, className }: IRenewDateCellProps) => {
  const formattedDate = monthDayYearFormat(value);

  // "w250" is a utility class that sets the width the the same as
  // the other text cells in the table.
  // TODO: consider creating a generic StatusIndicatorCell component that
  // take in a value, a desired status to display and, can contain these
  // table cell styles.
  const classNames = classnames(baseClass, className, "w250");

  let indicatorStatus: Exclude<IIndicatorValue, "indeterminate"> = "success";

  if (willExpireWithinXDays(value, 30)) {
    indicatorStatus = "warning";
  } else if (hasLicenseExpired(value)) {
    indicatorStatus = "error";
  }

  return (
    <StatusIndicator
      className={classNames}
      value={formattedDate}
      indicator={indicatorStatus}
    />
  );
};

export default RenewDateCell;
