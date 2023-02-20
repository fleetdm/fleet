import React from "react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITextCellProps {
  value: string | number | boolean | { timeString: string };
  formatter?: (val: any) => JSX.Element | string; // string, number, or null
  greyed?: boolean;
  classes?: string;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  greyed,
  classes = "w250",
}: ITextCellProps): JSX.Element => {
  let val = value;

  if (typeof value === "boolean") {
    val = value.toString();
  }
  return (
    <span className={`text-cell ${classes} ${greyed && "grey-cell"}`}>
      {formatter(val) || DEFAULT_EMPTY_CELL_VALUE}
    </span>
  );
};

export default TextCell;
