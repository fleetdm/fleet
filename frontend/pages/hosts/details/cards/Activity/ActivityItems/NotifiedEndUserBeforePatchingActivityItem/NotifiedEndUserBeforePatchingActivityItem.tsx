import React from "react";

import ActivityItem from "components/ActivityItem";
import { pluralize } from "utilities/strings/stringUtils";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

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
  const timeLabel = timeBefore === 300 ? "5 minutes" : "1 hour";
  const failed = status === "failed";
  const verb = failed ? "failed to notify" : "notified";

  // Bolds each title, uses an Oxford comma, and truncates past three titles
  // with a ", and N more app(s)" suffix (Figma dev note 5546:46306). Same
  // logic as the global feed template; host feed just drops the host name.
  const bold = (name: string) => <b>{getDisplayedSoftwareName(name)}</b>;
  const overflow = titles.length - 3;
  let titleList: React.ReactNode = null;
  if (titles.length === 1) {
    titleList = bold(titles[0]);
  } else if (titles.length === 2) {
    titleList = (
      <>
        {bold(titles[0])} and {bold(titles[1])}
      </>
    );
  } else if (titles.length === 3) {
    titleList = (
      <>
        {bold(titles[0])}, {bold(titles[1])}, and {bold(titles[2])}
      </>
    );
  } else if (titles.length > 3) {
    titleList = (
      <>
        {bold(titles[0])}, {bold(titles[1])}, {bold(titles[2])}, and {overflow}{" "}
        more {pluralize(overflow, "app")}
      </>
    );
  }

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      hideCancel={hideCancel}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
    >
      <b>Fleet</b> {verb} end user {timeLabel} before patching {titleList} on
      this host.
    </ActivityItem>
  );
};

export default NotifiedEndUserBeforePatchingActivityItem;
