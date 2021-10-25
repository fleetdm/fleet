import React from "react";
import classnames from "classnames";

interface IStatusCellProps {
  value: string;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const StatusCell = ({ value }: IStatusCellProps): JSX.Element => {
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${generateClassTag(value)}`
  );

  return <span className={statusClassName}>{value}</span>;
};

export default StatusCell;
