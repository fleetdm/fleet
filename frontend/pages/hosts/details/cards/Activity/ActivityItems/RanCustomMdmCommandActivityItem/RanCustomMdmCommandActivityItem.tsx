import React from "react";

import { formatMdmCommandNameForActivityItem } from "utilities/activityHelpers";
import ActivityItem from "components/ActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "ran-custom-mdm-command-activity-item";

const RanCustomMdmCommandActivityItem = ({
  activity,
  onShowDetails,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      onShowDetails={onShowDetails}
      isSoloActivity={isSoloActivity}
      hideCancel
    >
      <b>{activity.actor_full_name ?? "Fleet"}</b>
      {" ran "}
      {formatMdmCommandNameForActivityItem(activity.details?.request_type)}
      {" on this host."}
    </ActivityItem>
  );
};

export default RanCustomMdmCommandActivityItem;
