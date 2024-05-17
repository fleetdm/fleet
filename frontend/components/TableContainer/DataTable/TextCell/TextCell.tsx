import { uniqueId } from "lodash";
import React from "react";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITextCellProps {
  value?: string | number | boolean | { timeString: string } | null;
  formatter?: (val: any) => React.ReactNode; // string, number, or null
  greyed?: boolean;
  classes?: string;
  emptyCellTooltipText?: React.ReactNode;
}

const TextCell = ({
  value,
  formatter = (val) => val, // identity function if no formatter is provided
  greyed,
  classes = "w250",
  emptyCellTooltipText,
}: ITextCellProps): JSX.Element => {
  let val = value;

  if (typeof value === "boolean") {
    val = value.toString();
  }
  if (!val) {
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

  return (
    <span className={`text-cell ${classes} ${greyed ? "grey-cell" : ""}`}>
      {formatter(val) || renderEmptyCell()}
    </span>
  );
};

export default TextCell;
