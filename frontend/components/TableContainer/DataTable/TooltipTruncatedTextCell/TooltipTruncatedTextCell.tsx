import React from "react";
import { uniqueId } from "lodash";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { COLORS } from "styles/var/colors";

interface ITooltipTruncatedTextCellProps {
  value: React.ReactNode;
  /** Tooltip to dispay. If this is provided then this will be rendered as the tooltip content. If
   * not the value will be displayed as the tooltip content. Defaults to `undefined` */
  tooltip?: React.ReactNode;
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
  tooltip,
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
        className="data-table__tooltip-truncated-text"
        data-tip
        data-for={tooltipId}
        data-tip-disable={isDefaultValue}
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
        backgroundColor={COLORS["tooltip-bg"]}
        id={tooltipId}
        data-html
        className="truncated-tooltip" // responsive widths
        clickable
        delayHide={200} // need delay set to hover using clickable
      >
        <>
          {tooltip ?? value}
          <div className="safari-hack">&nbsp;</div>
          {/* Fixes triple click selecting next element in Safari */}
        </>
      </ReactTooltip>
    </div>
  );
};

export default TooltipTruncatedTextCell;
