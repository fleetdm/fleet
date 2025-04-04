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
        <b>{formatScriptNameForActivityItem(activity.details.script_name)}</b>{" "}
        script on this host.
      </>
    </ActivityItem>
  );
};

export default CanceledRunScriptActivityItem;
