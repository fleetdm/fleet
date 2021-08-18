import React from "react";

interface ITextCellProps {
  value: string | number | boolean;
  formatter?: (val: any) => string;
  greyed?: string;
}

const TextCell = (props: ITextCellProps): JSX.Element => {
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

export default TextCell;
