import React from "react";

import {
  ActivityType,
  IHostPastActivityType,
  IPastActivity,
} from "interfaces/activity";

import RanScriptActivityItem from "./RanScriptActivityItem";
import { ShowActivityDetailsHandler } from "./Activity";

interface IPastActivityFeedProps {
  activity: IPastActivity;
  onShowDetails?: ShowActivityDetailsHandler;
}

// eslint-disable-next-line import/prefer-default-export
export const pastActivityComponentMap: Record<
  IHostPastActivityType,
  React.FC<IPastActivityFeedProps>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.LockedHost]: RanScriptActivityItem,
  [ActivityType.UnlockedHost]: RanScriptActivityItem,
};
