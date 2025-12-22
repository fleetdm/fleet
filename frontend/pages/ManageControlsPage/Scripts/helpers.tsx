import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import Icon from "components/Icon";
import React from "react";
import { IScriptBatchSummaryV2 } from "services/entities/scripts";

import { isDateTimePast } from "utilities/helpers";

const getWhen = (summary: IScriptBatchSummaryV2) => {
  const {
    batch_execution_id: id,
    not_before,
    started_at,
    finished_at,
    canceled,
  } = summary;
  switch (summary.status) {
    case "started":
      if (!started_at || !isDateTimePast(started_at)) {
        console.warn(
          `Batch run with execution id ${id} is marked as 'started' but has no past 'started_at'`
        );
        return null;
      }
      return (
        <>
          <Icon name="pending-outline" color="ui-fleet-black-50" size="small" />
          Started{" "}
          <HumanTimeDiffWithFleetLaunchCutoff
            timeString={started_at}
            tooltipPosition="right"
          />
        </>
      );
    case "scheduled":
      if (!not_before || isDateTimePast(not_before)) {
        console.warn(
          `Batch run with execution id ${id} is marked as 'scheduled' but has no future scheduled start time`
        );
        return null;
      }
      return (
        <>
          <Icon name="clock" color="ui-fleet-black-50" size="small" />
          Will start{" "}
          <HumanTimeDiffWithFleetLaunchCutoff
            timeString={not_before}
            tooltipPosition="right"
          />
        </>
      );
    case "finished":
      if (!finished_at || !isDateTimePast(finished_at)) {
        console.warn(
          `Batch run with execution id ${id} is marked as 'finished' but has no past 'finished_at' data`
        );
        return null;
      }
      return (
        <>
          <Icon
            name={canceled ? "close-filled" : "success"}
            color="ui-fleet-black-50"
            size="small"
          />
          {canceled ? "Canceled" : "Completed"}
          <HumanTimeDiffWithFleetLaunchCutoff
            timeString={finished_at}
            tooltipPosition="right"
          />
        </>
      );
    default:
      return null;
  }
};

export default getWhen;
