import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import ActivityItem from "components/ActivityItem";

import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-run-script-activity-item";

const CanceledRunScriptActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel
      hideShowDetails
    >
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        {formatScriptNameForActivityItem(activity.details.script_name)} on this
        host.
      </>
    </ActivityItem>
  );
};

export default CanceledRunScriptActivityItem;
