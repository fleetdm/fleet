import React from "react";

import { IActivity, IActivityDetails } from "interfaces/activity";
import { IActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Button from "components/buttons/Button";
import DataError from "components/DataError";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import PastActivity from "../PastActivity/PastActivity";

const baseClass = "past-activity-feed";

interface IPastActivityFeedProps {
  activities?: IActivitiesResponse;
  isError?: boolean;
  onDetailsClick: (details: IActivityDetails) => void;
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
    return <DataError />;
  }

  if (!activities) {
    return null;
  }

  const { activities: activitiesList, meta } = activities;

  if (activitiesList.length === 0) {
    return (
      <EmptyFeed
        title="No Activity"
        message="When a script runs on a host, it shows up here."
      />
    );
  }

  return (
    <div className={baseClass}>
      {activitiesList.map((activity: IActivity) => (
        <PastActivity activity={activity} onDetailsClick={onDetailsClick} />
      ))}
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
