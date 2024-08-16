import React from "react";

import StatusIndicator from "components/StatusIndicator";

import { monthDayYearFormat } from "utilities/date_format";

const baseClass = "renew-date-cell";

interface IRenewDateCellProps {
  value: string;
}

const RenewDateCell = ({ value }: IRenewDateCellProps) => {
  const formattedDate = monthDayYearFormat(value);

  let indicatorStatus = "success";

  return (
    <StatusIndicator
      className={baseClass}
      value={formattedDate}
      indicator="success"
    />
  );
};

export default RenewDateCell;
