import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import HostActivityItem from "../../HostActivityItem";
import { IHostActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-script-activity-item";

const CanceledScriptActivityItem = ({
  activity,
}: IHostActivityItemComponentProps) => {
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>{formatScriptNameForActivityItem(activity.details?.script_name)}</b>{" "}
        script on this host.
      </>
    </HostActivityItem>
  );
};

export default CanceledScriptActivityItem;
