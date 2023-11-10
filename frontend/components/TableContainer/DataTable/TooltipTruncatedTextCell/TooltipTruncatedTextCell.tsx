import React from "react";
import { uniqueId } from "lodash";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITooltipTruncatedTextCellProps {
  value: string | number | boolean;
  /** If set to `true` the text inside the tooltip will break on words instead of any character.
   * By default the tooltip text breaks on any character.
   * Default is `false`.
   */
  tooltipBreakOnWord?: boolean;
  classes?: string;
}

const baseClass = "tooltip-truncated-cell";

const TooltipTruncatedTextCell = ({
  value,
  tooltipBreakOnWord = false,
  classes = "w250",
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  const classNames = classnames(baseClass, classes, {
    "tooltip-break-on-word": tooltipBreakOnWord,
  });

  const tooltipId = uniqueId();
  const isDefaultValue = value === DEFAULT_EMPTY_CELL_VALUE;
  return (
    <div className={classNames}>
      <div
        className={"data-table__tooltip-truncated-text"}
        data-tip
        data-for={tooltipId}
      >
        <span
          className={`data-table__tooltip-truncated-text--cell ${
            isDefaultValue ? "text-muted" : ""
          } `}
        >
          {value}
        </span>
      </div>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor="#3e4771"
        id={tooltipId}
        data-html
        className={"truncated-tooltip"} // responsive widths
        clickable
        delayHide={200} // need delay set to hover using clickable
      >
        <>
          {value}
          <div className="safari-hack">&nbsp;</div>
          {/* Fixes triple click selecting next element in Safari */}
        </>
      </ReactTooltip>
    </div>
  );
};

export default TooltipTruncatedTextCell;
