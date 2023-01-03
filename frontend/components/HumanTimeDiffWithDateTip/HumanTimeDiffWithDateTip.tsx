import React from "react";

import { uniqueId } from "lodash";
import { humanHostLastSeen } from "utilities/helpers";
import ReactTooltip from "react-tooltip";
import intlFormat from "date-fns/intlFormat";

// TODO - timeString needs to be any for certain places this is being used
// Once those usese of 'any' are improved, this can be of type 'string'
export default (timeString: string | any): JSX.Element => {
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
