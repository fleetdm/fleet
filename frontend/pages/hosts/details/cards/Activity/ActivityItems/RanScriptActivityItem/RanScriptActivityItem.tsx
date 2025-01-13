import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import HostActivityItem from "../../../../../../../components/ActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "ran-script-activity-item";

const RanScriptActivityItem = ({
  tab,
  activity,
  onShowDetails,
  onCancel,
  soloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const ranScriptPrefix = tab === "past" ? "ran" : "told Fleet to run";

  return (
    <HostActivityItem
      className={baseClass}
      activity={activity}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      soloActivity={soloActivity}
    >
      <b>{activity.actor_full_name}</b>
      <>
        {" "}
        {ranScriptPrefix}{" "}
        {formatScriptNameForActivityItem(activity.details?.script_name)} on this
        host.{" "}
      </>
    </HostActivityItem>
  );
};

export default RanScriptActivityItem;
