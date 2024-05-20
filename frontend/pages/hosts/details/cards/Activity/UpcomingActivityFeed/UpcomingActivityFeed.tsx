import React from "react";

import { IHostActivity } from "interfaces/activity";
import { IUpcomingActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import DataError from "components/DataError";
import Button from "components/buttons/Button";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import UpcomingActivity from "../UpcomingActivity/UpcomingActivity";
import { ShowActivityDetailsHandler } from "../Activity";

const baseClass = "upcoming-activity-feed";

interface IUpcomingActivityFeedProps {
  activities?: IUpcomingActivitiesResponse;
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
        message="When you run a script on an offline host, it will appear here."
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div>
        {activitiesList.map((activity: IHostActivity) => (
          <UpcomingActivity
            key={activity.id}
            activity={activity}
            onDetailsClick={onDetailsClick}
          />
        ))}
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
