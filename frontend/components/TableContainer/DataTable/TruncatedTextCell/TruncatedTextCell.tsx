import React, { useState, useRef, useLayoutEffect } from "react";
import { uniqueId } from "lodash";

import ReactTooltip from "react-tooltip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITruncatedTextCellProps {
  value: string | number | boolean;
  classes?: string;
}

const baseClass = "truncated-cell";

const TruncatedTextCell = ({
  value,
  classes = "w250",
}: ITruncatedTextCellProps): JSX.Element => {
  const ref = useRef<HTMLInputElement>(null);

  const [offsetWidth, setOffsetWidth] = useState(0);
  const [scrollWidth, setScrollWidth] = useState(0);

  useLayoutEffect(() => {
    if (ref?.current !== null) {
      setOffsetWidth(ref.current.offsetWidth);
      setScrollWidth(ref.current.scrollWidth);
    }
  }, []);

  const tooltipId = uniqueId();
  const tooltipDisabled = offsetWidth === scrollWidth;
  const isDefaultValue = value === DEFAULT_EMPTY_CELL_VALUE;
  return (
    <div ref={ref} className={`${baseClass} ${classes}`}>
      <div
        className={"data-table__truncated-text"}
        data-tip
        data-for={tooltipId}
        data-tip-disable={tooltipDisabled}
      >
        <span
          className={`data-table__truncated-text--cell ${
            isDefaultValue ? "text-muted" : ""
          } ${tooltipDisabled ? "" : "truncated"}`}
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

export default TruncatedTextCell;
