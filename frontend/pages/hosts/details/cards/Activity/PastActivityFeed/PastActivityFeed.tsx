import React from "react";

import { ActivityType, IHostPastActivity } from "interfaces/activity";
import { IHostPastActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Button from "components/buttons/Button";
import DataError from "components/DataError";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import { ShowActivityDetailsHandler } from "../Activity";

import { pastActivityComponentMap } from "../ActivityConfig";

const baseClass = "past-activity-feed";

interface IPastActivityFeedProps {
  activities?: IHostPastActivitiesResponse;
  isError?: boolean;
  onDetailsClick: ShowActivityDetailsHandler;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const PastActivityFeed = ({
  activities,
  isError = false,
  onDetailsClick,
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
          const ActivityItemComponent = pastActivityComponentMap[activity.type];
          return (
            <ActivityItemComponent
              key={activity.id}
              tab="past"
              activity={activity}
              hideClose
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

export default PastActivityFeed;
