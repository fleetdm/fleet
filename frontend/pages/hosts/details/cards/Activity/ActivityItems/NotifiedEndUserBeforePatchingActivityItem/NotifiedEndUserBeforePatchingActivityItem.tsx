import React from "react";

import { INotifyActivityStatus } from "interfaces/activity";
import ActivityItem from "components/ActivityItem";
import {
  renderNotifyTitleList,
  formatNotifyTimeLabel,
  isNotifyFailure,
} from "components/ActivityDetails/NotifyBeforePatchingDetailsModal/helpers";

import { IHostActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "notified-end-user-before-patching-activity-item";

const NotifiedEndUserBeforePatchingActivityItem = ({
  activity,
  onShowDetails,
  onCancel,
  hideCancel,
  isSoloActivity,
}: IHostActivityItemComponentPropsWithShowDetails) => {
  const { details } = activity;
  const {
    software_titles: titles = [],
    status,
    time_before: timeBefore,
  } = details;
  const timeLabel = formatNotifyTimeLabel(timeBefore);
  const failed = isNotifyFailure(status as INotifyActivityStatus | undefined);
  const verb = failed ? "failed to notify" : "notified";
  const titleList = renderNotifyTitleList(titles);

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
    >
      <strong>Fleet</strong> {verb} end user {timeLabel} before patching{" "}
      {titleList} on this host.
    </ActivityItem>
  );
};

export default NotifiedEndUserBeforePatchingActivityItem;
