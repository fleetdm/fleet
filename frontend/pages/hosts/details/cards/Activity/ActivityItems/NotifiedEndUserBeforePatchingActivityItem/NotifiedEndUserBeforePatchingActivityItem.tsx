import React from "react";

import {
  renderNotifyTitleList,
  formatNotifyTimeLabel,
  isNotifyFailure,
} from "components/ActivityDetails/NotifyBeforePatchingDetailsModal/helpers";
import ActivityItem from "components/ActivityItem";

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
    software_titles: legacyTitles = [],
    software,
    status,
    time_before: timeBefore,
  } = details;
  const timeLabel = formatNotifyTimeLabel(timeBefore);
  const failed = isNotifyFailure(status);
  const verb = failed ? "failed to notify" : "notified";
  // Prefer the richer `software` field on new activities; fall back to the
  // legacy flat `software_titles` for pre-override activity rows.
  const titleList = renderNotifyTitleList(software ?? legacyTitles);

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
    >
      <strong>Fleet</strong> {verb} end user {timeLabel} before patching
      {titleList && <> {titleList}</>} on this host.
    </ActivityItem>
  );
};

export default NotifiedEndUserBeforePatchingActivityItem;
