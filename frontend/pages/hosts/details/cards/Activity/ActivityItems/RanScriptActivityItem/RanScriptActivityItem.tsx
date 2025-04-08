import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "ran-script-activity-item";

const RanScriptActivityItem = ({
  tab,
  activity,
  onShowDetails,
  onCancel,
  isSoloActivity,
  hideCancel,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  let ranScriptPrefix = tab === "past" ? "ran" : "told Fleet to run";
  if (tab !== "past" && activity.fleet_initiated) {
    ranScriptPrefix = "will run";
  }

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
      hideCancel={hideCancel}
    >
      <b>{activity.actor_full_name ?? "Fleet"}</b>
      <>
        {" "}
        {ranScriptPrefix}{" "}
        {formatScriptNameForActivityItem(activity.details?.script_name)} on this
        host.{" "}
      </>
    </ActivityItem>
  );
};

export default RanScriptActivityItem;
