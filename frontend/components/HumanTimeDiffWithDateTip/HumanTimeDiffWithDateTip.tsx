import React from "react";

import { humanLastSeen, internationalTimeFormat } from "utilities/helpers";
import { INITIAL_FLEET_DATE } from "utilities/constants";
import TooltipWrapper from "components/TooltipWrapper";
import { PlacesType } from "react-tooltip-5";

interface IHumanTimeDiffWithDateTip {
  timeString: string;
  cutoffBeforeFleetLaunch?: boolean;
  tooltipPosition?: PlacesType;
}

/** Returns "Unavailable" if date is empty string or "Unavailable"
 * Returns "Invalid date" if date is invalid
 * Returns "Never" if cutoffBeforeFleetLaunch is true and date is before the
 * initial launch of Fleet */
export const HumanTimeDiffWithDateTip = ({
  timeString,
  cutoffBeforeFleetLaunch = false,
  tooltipPosition = "top",
}: IHumanTimeDiffWithDateTip): JSX.Element => {
  if (timeString === "Unavailable" || timeString === "") {
    return <span>Unavailable</span>;
  }

  // There are cases where dates are set in Fleet to be the "zero date" which
  // serves as an indicator that a particular date isn't set.
  if (cutoffBeforeFleetLaunch && timeString < INITIAL_FLEET_DATE) {
    return <span>Never</span>;
  }

  try {
    return (
      <TooltipWrapper
        className="date-tooltip"
        tooltipClass="date-tooltip-text"
        tipContent={internationalTimeFormat(new Date(timeString))}
        position={tooltipPosition}
        underline={false}
      >
        {humanLastSeen(timeString)}
      </TooltipWrapper>
    );
  } catch (e) {
    if (e instanceof RangeError) {
      return <span>Invalid date</span>;
    }
    return <span>Unavailable</span>;
  }
};

/** Returns a HumanTimeDiffWithDateTip configured to return "Never" in the case
 * that the timeString is before the launch date of Fleet */
export const HumanTimeDiffWithFleetLaunchCutoff = ({
  timeString,
  tooltipPosition = "top",
}: IHumanTimeDiffWithDateTip): JSX.Element => {
  return (
    <HumanTimeDiffWithDateTip
      timeString={timeString}
      tooltipPosition={tooltipPosition}
      cutoffBeforeFleetLaunch
    />
  );
};
