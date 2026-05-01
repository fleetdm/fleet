import classnames from "classnames";
import React from "react";
import TooltipWrapper from "components/TooltipWrapper";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "text-cell";
interface ITextCellProps {
  value?: React.ReactNode | { timeString: string };
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  formatter?: (val: any) => React.ReactNode; // string, number, or null
  grey?: boolean;
  italic?: boolean;
  className?: string;
  emptyCellTooltipText?: React.ReactNode;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  grey = false,
  italic = false,
  className = "w250",
  emptyCellTooltipText,
}: ITextCellProps) => {
  let val = value;

  // we want to render booleans as strings.
  if (typeof value === "boolean") {
    val = value.toString();
  }

  const formattedValue = formatter(val);

  // Check if the given value is empty or if the formatted value is empty.
  // 'empty' is defined as null, undefined, or an empty string.
  const isEmptyValue =
    value === null ||
    value === undefined ||
    value === "" ||
    formattedValue === null ||
    formattedValue === undefined ||
    formattedValue === "";

  if (isEmptyValue) {
    [grey, italic] = [true, true];
  }

  const renderEmptyCell = () => {
    if (emptyCellTooltipText) {
      return (
        <TooltipWrapper
          tipContent={emptyCellTooltipText}
          position="top"
          underline={false}
          showArrow
        >
          <span>{DEFAULT_EMPTY_CELL_VALUE}</span>
        </TooltipWrapper>
      );
    }
    return DEFAULT_EMPTY_CELL_VALUE;
  };

  const cellText = isEmptyValue ? renderEmptyCell() : formattedValue;

  const cellClasses = classnames(baseClass, className, {
    "grey-cell": grey,
    "italic-cell": italic,
  });
  return <span className={cellClasses}>{cellText}</span>;
};

export default TextCell;
