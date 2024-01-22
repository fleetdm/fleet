import React from "react";

import { IActivityDetails } from "interfaces/activity";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import UpcomingActivity from "../UpcomingActivity/UpcomingActivity";

const baseClass = "upcoming-activity-feed";

interface IUpcomingActivityFeedProps {
  activities: any; // TODO: type
  onDetailsClick: (details: IActivityDetails) => void;
}

const testActivity = {
  created_at: "2021-07-27T13:25:21Z",
  id: 4,
  actor_full_name: "Rachael",
  actor_id: 1,
  actor_gravatar: "",
  actor_email: "rachael@example.com",
  type: "ran_script",
  details: {
    host_id: 1,
    host_display_name: "Steve's MacBook Pro",
    script_name: "",
    script_execution_id: "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
  },
};

const UpcomingActivityFeed = ({
  activities,
  onDetailsClick,
}: IUpcomingActivityFeedProps) => {
  activities = [testActivity];

  if (activities.length === 0) {
    return (
      <EmptyFeed
        title="No pending activity "
        message="When you run a script on an offline host, it will appear here."
      />
    );
  }

  return (
    <div className={baseClass}>
      {activities?.map((activity: any) => (
        <UpcomingActivity activity={activity} onDetailsClick={onDetailsClick} />
      ))}
    </div>
  );
};

export default UpcomingActivityFeed;
