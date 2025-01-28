import classnames from "classnames";
import { uniqueId } from "lodash";
import React from "react";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "text-cell";
interface ITextCellProps {
  value?: React.ReactNode | { timeString: string };
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
      const tooltipId = uniqueId();
      return (
        <>
          <span data-tip data-for={tooltipId}>
            {DEFAULT_EMPTY_CELL_VALUE}
          </span>
          <ReactTooltip
            place="top"
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id={tooltipId}
          >
            {emptyCellTooltipText}
          </ReactTooltip>
        </>
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
