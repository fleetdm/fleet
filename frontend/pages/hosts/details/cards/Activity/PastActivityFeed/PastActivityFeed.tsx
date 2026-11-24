import React, { useContext } from "react";

import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";
import DataError from "components/DataError";
import Pagination from "components/Pagination";
import { AppContext } from "context/app";
import { IHostPastActivity } from "interfaces/activity";
import { IHostPastActivitiesResponse } from "services/entities/activities";
import { PREMIUM_ONLY_DETAIL_ACTIVITIES } from "utilities/activityHelpers";

import { pastActivityComponentMap } from "../ActivityConfig";
import EmptyFeed from "../EmptyFeed/EmptyFeed";

const baseClass = "past-activity-feed";

interface IPastActivityFeedProps {
  activities?: IHostPastActivitiesResponse;
  isError?: boolean;
  onShowDetails: ShowActivityDetailsHandler;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const PastActivityFeed = ({
  activities,
  isError = false,
  onShowDetails,
  onNextPage,
  onPreviousPage,
}: IPastActivityFeedProps) => {
  const { isPremiumTier } = useContext(AppContext);

  if (isError) {
    return <DataError verticalPaddingSize="pad-large" />;
  }

  if (!activities) {
    return null;
  }

  const { activities: activitiesList, meta } = activities;

  if (activitiesList === null || activitiesList.length === 0) {
    return (
      <EmptyFeed
        title="No activity"
        message={
          isPremiumTier
            ? "Completed commands (e.g. lock, wipe) will appear here."
            : "Completed commands will appear here."
        }
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div>
        {activitiesList.map((activity: IHostPastActivity) => {
          const ActivityItemComponent = pastActivityComponentMap[activity.type];
          if (!ActivityItemComponent) {
            // Log so we catch missing frontend registrations, but don't crash the page.
            // eslint-disable-next-line no-console
            console.warn(
              `No PastActivityFeed component registered for activity type: ${activity.type}`
            );
            return null;
          }
          const hideShowDetails =
            !isPremiumTier && PREMIUM_ONLY_DETAIL_ACTIVITIES.has(activity.type);
          return (
            <ActivityItemComponent
              key={activity.id}
              tab="past"
              activity={activity}
              hideCancel
              hideShowDetails={hideShowDetails}
              onShowDetails={onShowDetails}
            />
          );
        })}
      </div>
      <Pagination
        disablePrev={!meta.has_previous_results}
        disableNext={!meta.has_next_results}
        hidePagination={!meta.has_next_results && !meta.has_previous_results}
        onPrevPage={onPreviousPage}
        onNextPage={onNextPage}
      />
    </div>
  );
};

export default PastActivityFeed;
