import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import HostActivityItem from "../../HostActivityItem";
import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";
import ShowDetailsButton from "../../ShowDetailsButton";

const baseClass = "ran-script-activity-item";

const RanScriptActivityItem = ({
  tab,
  activity,
  onShowDetails,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const ranScriptPrefix = tab === "past" ? "ran" : "told Fleet to run";

  return (
    <HostActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b>
      <>
        {" "}
        {ranScriptPrefix}{" "}
        {formatScriptNameForActivityItem(activity.details?.script_name)} on this
        host.{" "}
        {/* <ShowDetailsButton activity={activity} onShowDetails={onShowDetails} /> */}
      </>
    </HostActivityItem>
  );
};

export default RanScriptActivityItem;
