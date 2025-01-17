import React from "react";

import { ActivityType, IHostPastActivity } from "interfaces/activity";
import { IHostPastActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import EmptyFeed from "../EmptyFeed/EmptyFeed";

import { pastActivityComponentMap } from "../ActivityConfig";

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
  if (isError) {
    return <DataError className={`${baseClass}__error`} />;
  }

  if (!activities) {
    return null;
  }

  const { activities: activitiesList, meta } = activities;

  if (activitiesList === null || activitiesList.length === 0) {
    return (
      <EmptyFeed
        title="No activity"
        message="Completed actions will appear here (scripts, software, lock, and wipe)."
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div>
        {activitiesList.map((activity: IHostPastActivity) => {
          // TODO: remove this once we have a proper way of handling "Fleet-initiated" activities in
          // the backend. For now, if all these fields are empty, then we assume it was
          // Fleet-initiated.
          if (
            !activity.actor_email &&
            !activity.actor_full_name &&
            (activity.type === ActivityType.InstalledSoftware ||
              activity.type === ActivityType.InstalledAppStoreApp ||
              activity.type === ActivityType.RanScript)
          ) {
            activity.actor_full_name = "Fleet";
          }
          const ActivityItemComponent = pastActivityComponentMap[activity.type];
          return (
            <ActivityItemComponent
              key={activity.id}
              tab="past"
              activity={activity}
              hideCancel
              onShowDetails={onShowDetails}
            />
          );
        })}
      </div>
      <div className={`${baseClass}__pagination`}>
        <Button
          disabled={!meta.has_previous_results}
          onClick={onPreviousPage}
          variant="unstyled"
          className={`${baseClass}__load-activities-button`}
        >
          <>
            <FleetIcon name="chevronleft" /> Previous
          </>
        </Button>
        <Button
          disabled={!meta.has_next_results}
          onClick={onNextPage}
          variant="unstyled"
          className={`${baseClass}__load-activities-button`}
        >
          <>
            Next <FleetIcon name="chevronright" />
          </>
        </Button>
      </div>
    </div>
  );
};

export default PastActivityFeed;
