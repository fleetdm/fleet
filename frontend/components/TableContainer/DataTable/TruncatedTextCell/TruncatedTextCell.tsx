import React, { useState, useRef, useLayoutEffect } from "react";
import { v4 as uuidv4 } from "uuid";

import ReactTooltip from "react-tooltip";

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

  const id = uuidv4();
  const tooltipDisabled = offsetWidth === scrollWidth;

  return (
    <div ref={ref} className={`${baseClass} ${classes}`}>
      <div
        className={"data-table__truncated-text"}
        data-tip
        data-for={id}
        data-tip-disable={tooltipDisabled}
      >
        <span
          className={`data-table__truncated-text--cell ${
            tooltipDisabled ? "" : "truncated"
          }`}
        >
          {value}
        </span>
      </div>
      <ReactTooltip
        place="bottom"
        effect="solid"
        backgroundColor="#3e4771"
        id={id}
        data-html
        className={"truncated-tooltip"} // responsive widths
      >
        {value}
      </ReactTooltip>
    </div>
  );
};

export default TruncatedTextCell;
