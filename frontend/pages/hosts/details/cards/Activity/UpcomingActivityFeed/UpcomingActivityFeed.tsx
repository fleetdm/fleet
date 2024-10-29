import React from "react";

import { IHostUpcomingActivity } from "interfaces/activity";
import { IHostUpcomingActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import DataError from "components/DataError";
import Button from "components/buttons/Button";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import { ShowActivityDetailsHandler } from "../Activity";
import { upcomingActivityComponentMap } from "../ActivityConfig";

const baseClass = "upcoming-activity-feed";

interface IUpcomingActivityFeedProps {
  activities?: IHostUpcomingActivitiesResponse;
  isError?: boolean;
  onDetailsClick: ShowActivityDetailsHandler;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const UpcomingActivityFeed = ({
  activities,
  isError = false,
  onDetailsClick,
  onNextPage,
  onPreviousPage,
}: IUpcomingActivityFeedProps) => {
  if (isError) {
    return <DataError />;
  }

  if (!activities) {
    return null;
  }

  const { activities: activitiesList, meta } = activities;

  if (activitiesList === null || activitiesList.length === 0) {
    return (
      <EmptyFeed
        title="No pending activity "
        message="Pending actions will appear here (scripts, software, lock, and wipe)."
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div>
        {activitiesList.map((activity: IHostUpcomingActivity) => {
          // TODO: remove this once we have a proper way of handling "Fleet-initiated" activities in
          // the backend. For now, if all these fields are empty, then we assume it was Fleet-initiated.
          if (
            !activity.actor_email &&
            !activity.actor_full_name &&
            !activity.actor_id
          ) {
            activity.actor_full_name = "Fleet";
          }
          const ActivityItemComponent =
            upcomingActivityComponentMap[activity.type];
          return (
            <ActivityItemComponent
              key={activity.id}
              tab="upcoming"
              activity={activity}
              onShowDetails={onDetailsClick}
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

export default UpcomingActivityFeed;
