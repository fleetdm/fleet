import React from "react";
import classnames from "classnames";

interface IStatusCellProps {
  value: string;
}

const StatusCell = (props: IStatusCellProps): JSX.Element => {
  const { value } = props;
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${value}`
  );

  return <span className={statusClassName}>{value}</span>;
};

export default StatusCell;
