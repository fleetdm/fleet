import React from "react";

import { IActivity, IPastActivity } from "interfaces/activity";
import { formatScriptNameForActivityItem } from "utilities/helpers";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

import HostActivityItem from "../HostActivityItem";
import { IHostActivityItemComponentProps } from "../ActivityConfig";

const baseClass = "ran-script-activity-item";

const RanScriptActivityItem = ({
  activity,
  onShowDetails,
}: IHostActivityItemComponentProps) => {
  return (
    <HostActivityItem activity={activity}>
      <b>{activity.actor_full_name}</b>
      <>
        {" "}
        ran {formatScriptNameForActivityItem(activity.details?.script_name)} on
        this host.{" "}
        <Button
          className={`${baseClass}__show-query-link`}
          variant="text-link"
          onClick={() => onShowDetails?.(activity)}
        >
          Show details{" "}
          <Icon className={`${baseClass}__show-query-icon`} name="eye" />
        </Button>
      </>
    </HostActivityItem>
  );
};

export default RanScriptActivityItem;
