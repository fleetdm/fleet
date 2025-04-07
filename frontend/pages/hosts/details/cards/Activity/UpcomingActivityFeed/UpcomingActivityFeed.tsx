import React, { useContext } from "react";

import { IHostUpcomingActivity } from "interfaces/activity";
import { IHostUpcomingActivitiesResponse } from "services/entities/activities";
import { AppContext } from "context/app";

import DataError from "components/DataError";
import Pagination from "components/Pagination";
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
  const {
    isTeamMaintainerOrTeamAdmin,
    isGlobalAdmin,
    isGlobalMaintainer,
  } = useContext(AppContext);

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

  const canCancel =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainerOrTeamAdmin;

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
              hideCancel={!canCancel}
              onCancel={() => onCancel(activity)}
            />
          );
        })}
      </div>
      <Pagination
        disablePrev={!meta.has_previous_results}
        disableNext={!meta.has_next_results}
        hidePagination={!meta.has_previous_results && !meta.has_next_results}
        onPrevPage={onPreviousPage}
        onNextPage={onNextPage}
      />
    </div>
  );
};

export default UpcomingActivityFeed;
