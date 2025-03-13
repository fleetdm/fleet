import React from "react";

import { IHostUpcomingActivity } from "interfaces/activity";
import { IHostUpcomingActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import { upcomingActivityComponentMap } from "../ActivityConfig";

const baseClass = "upcoming-activity-feed";

interface IUpcomingActivityFeedProps {
  activities?: IHostUpcomingActivitiesResponse;
  isError?: boolean;
  onShowDetails: ShowActivityDetailsHandler;
  onCancel: (activity: IHostUpcomingActivity) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const UpcomingActivityFeed = ({
  activities,
  isError = false,
  onShowDetails,
  onCancel,
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
      <div className={`${baseClass}__feed-list`}>
        {activitiesList.map((activity: IHostUpcomingActivity) => {
          const ActivityItemComponent =
            upcomingActivityComponentMap[activity.type];
          return (
            <ActivityItemComponent
              key={activity.id}
              tab="upcoming"
              activity={activity}
              onShowDetails={onShowDetails}
              hideCancel // TODO: remove this when canceling is implemented in API
              onCancel={() => onCancel(activity)}
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
