import React, { useRef, useState } from "react";

import { ActivityType, IActivityDetails } from "interfaces/activity";
import { getPerformanceImpactDescription } from "utilities/helpers";
import ActivityItem from "pages/DashboardPage/cards/ActivityFeed/ActivityItem";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import PastActivity from "../PastActivity/PastActivity";

const baseClass = "past-activity-feed";

interface IPastActivityFeedProps {
  activities: any; // TODO: type
  onDetailsClick: (details: IActivityDetails) => void;
}

const testActivity = {
  created_at: "2021-07-27T13:25:21Z",
  id: 1,
  actor_full_name: "Bob",
  actor_id: 2,
  actor_gravatar: "",
  actor_email: "bob@example.com",
  type: "ran_script",
  details: {
    host_id: 1,
    host_display_name: "Steve's MacBook Pro",
    script_name: "",
    script_execution_id: "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
  },
};

const PastActivityFeed = ({
  activities,
  onDetailsClick,
}: IPastActivityFeedProps) => {
  activities = [testActivity];

  const [pageIndex, setPageIndex] = useState(0);
  const [showShowQueryModal, setShowShowQueryModal] = useState(false);
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  const queryShown = useRef("");
  const queryImpact = useRef<string | undefined>(undefined);
  const scriptExecutionId = useRef("");

  const handleDetailsClick = (
    activityType: ActivityType,
    details: IActivityDetails
  ) => {
    switch (activityType) {
      case ActivityType.LiveQuery:
        queryShown.current = details.query_sql ?? "";
        queryImpact.current = details.stats
          ? getPerformanceImpactDescription(details.stats)
          : undefined;
        setShowShowQueryModal(true);
        break;
      case ActivityType.RanScript:
        scriptExecutionId.current = details.script_execution_id ?? "";
        setShowScriptDetailsModal(true);
        break;
      default:
        break;
    }
  };

  if (activities.length === 0) {
    return (
      <EmptyFeed
        title="No Activity"
        message="When a script runs on a host, it shows up here."
      />
    );
  }

  return (
    <div className={baseClass}>
      {activities?.map((activity: any) => (
        <PastActivity activity={activity} onDetailsClick={onDetailsClick} />
      ))}
    </div>
  );
};

export default PastActivityFeed;
