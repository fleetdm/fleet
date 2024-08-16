import React from "react";

import StatusIndicator from "components/StatusIndicator";

import { monthDayYearFormat } from "utilities/date_format";
import { hasLicenseExpired, willExpireWithinXDays } from "utilities/helpers";
import { IIndicatorValue } from "components/StatusIndicator/StatusIndicator";

const baseClass = "renew-date-cell";

interface IRenewDateCellProps {
  value: string;
}

const RenewDateCell = ({ value }: IRenewDateCellProps) => {
  const formattedDate = monthDayYearFormat(value);

  let indicatorStatus: Exclude<IIndicatorValue, "indeterminate"> = "success";

  if (willExpireWithinXDays(value, 30)) {
    indicatorStatus = "warning";
  } else if (hasLicenseExpired(value)) {
    indicatorStatus = "error";
  }

  return (
    <StatusIndicator
      className={baseClass}
      value={formattedDate}
      indicator={indicatorStatus}
    />
  );
};

export default RenewDateCell;
