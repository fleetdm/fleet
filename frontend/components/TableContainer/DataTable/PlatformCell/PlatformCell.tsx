import React from "react";

// TODO Implement platform icons logic within cell based on array of platform strings

interface IPlatformCellProps {
  value: string | number | boolean;
  formatter?: (val: any) => string;
  greyed?: string;
}

const PlatformCell = (props: IPlatformCellProps): JSX.Element => {
  const {
    value,
    formatter = (val) => val, // identity function if no formatter is provided
    greyed,
  } = props;

  let val = value;

  if (typeof value === "boolean") {
    val = value.toString();
  }

  return <span className={greyed}>{formatter(val)}</span>;
};

export default PlatformCell;
