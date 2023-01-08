import React from "react";

import { uniqueId } from "lodash";
import { humanHostLastSeen } from "utilities/helpers";
import ReactTooltip from "react-tooltip";
import intlFormat from "date-fns/intlFormat";

export default ({ timeString }: { timeString: string }): JSX.Element => {
  const id = uniqueId();
  return timeString === "Unavailable" ? (
    <span>Unavailable</span>
  ) : (
    <>
      <span className={"date-tooltip"} data-tip data-for={`tooltip-${id}`}>
        {humanHostLastSeen(timeString)}
      </span>
      <ReactTooltip
        className="date-tooltip-text"
        place="top"
        type="dark"
        effect="solid"
        id={`tooltip-${id}`}
        backgroundColor="#3e4771"
      >
        {intlFormat(
          new Date(timeString),
          {
            year: "numeric",
            month: "numeric",
            day: "numeric",
            hour: "numeric",
            minute: "numeric",
            second: "numeric",
          },
          { locale: window.navigator.languages[0] }
        )}
      </ReactTooltip>
    </>
  );
};
