import { uniqueId } from "lodash";
import React from "react";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITextCellProps {
  value?: string | number | boolean | { timeString: string } | null;
  formatter?: (val: any) => React.ReactNode; // string, number, or null
  /** adds a greyed styling to the cell. This will italicise and add a grey
   * color to the cell text.
   * @default false
   */
  greyed?: boolean;
  classes?: string;
  emptyCellTooltipText?: React.ReactNode;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  greyed = false,
  classes = "w250",
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
    greyed = true;
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

  return (
    <span className={`text-cell ${classes} ${greyed ? "grey-cell" : ""}`}>
      {cellText}
    </span>
  );
};

export default TextCell;
