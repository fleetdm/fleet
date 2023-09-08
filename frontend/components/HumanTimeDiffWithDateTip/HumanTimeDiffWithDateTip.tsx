import React from "react";

import { uniqueId } from "lodash";
import { humanLastSeen, internationalTimeFormat } from "utilities/helpers";
import ReactTooltip from "react-tooltip";

interface IHumanTimeDiffWithDateTip {
  timeString: string;
}

/** Returns Unavailable if date is "Unavailable" or empty string
 * Returns "Invalid date" if date is invalid */
export default ({ timeString }: IHumanTimeDiffWithDateTip): JSX.Element => {
  const id = uniqueId();

  if (timeString === "Unavailable" || timeString === "") {
    return <span>Unavailable</span>;
  }

  try {
    return (
      <>
        <span className={"date-tooltip"} data-tip data-for={`tooltip-${id}`}>
          {humanLastSeen(timeString)}
        </span>
        <ReactTooltip
          className="date-tooltip-text"
          place="top"
          type="dark"
          effect="solid"
          id={`tooltip-${id}`}
          backgroundColor="#3e4771"
        >
          {internationalTimeFormat(new Date(timeString))}
        </ReactTooltip>
      </>
    );
  } catch (e) {
    if (e instanceof RangeError) {
      return <span>Invalid date</span>;
    }
    return <span>Unavailable</span>;
  }
};
