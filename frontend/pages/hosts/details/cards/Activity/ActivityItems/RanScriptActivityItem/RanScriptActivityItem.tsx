import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import HostActivityItem from "../../HostActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "ran-script-activity-item";

const RanScriptActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b>
      <>
        {" "}
        ran {formatScriptNameForActivityItem(activity.details?.script_name)} on
        this host.{" "}
        <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} />
      </>
    </HostActivityItem>
  );
};

export default RanScriptActivityItem;
