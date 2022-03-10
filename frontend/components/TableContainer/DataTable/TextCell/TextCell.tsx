import React from "react";

interface ITextCellProps {
  value: string | number | boolean;
  formatter?: (val: any) => string; // string, number, or null
  greyed?: string;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  greyed,
}: ITextCellProps): JSX.Element => {
  let val = value;

  if (typeof value === "boolean") {
    val = value.toString();
  }

  return <span className={greyed}>{formatter(val)}</span>;
};

export default TextCell;
