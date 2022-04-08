import React from "react";

interface ITextCellProps {
  value: string | number | boolean;
  formatter?: (val: any) => string; // string, number, or null
  greyed?: string;
  classes?: string;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  greyed,
  classes: className = "w250",
}: ITextCellProps): JSX.Element => {
  let val = value;

  if (typeof value === "boolean") {
    val = value.toString();
  }

  return (
    <span className={`text-cell ${className} ${greyed || ""}`}>
      {formatter(val)}
    </span>
  );
};

export default TextCell;
