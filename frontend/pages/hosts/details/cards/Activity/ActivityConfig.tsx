import React from "react";

import {
  ActivityType,
  IHostPastActivityType,
  IPastActivity,
} from "interfaces/activity";

import { ShowActivityDetailsHandler } from "./Activity";

import RanScriptActivityItem from "./ActivityItems/RanScriptActivityItem";
import LockedHostActivityItem from "./ActivityItems/LockedHostActivityItem";
import UnlockedHostActivityItem from "./ActivityItems/UnlockedHostActivityItem";

/** the component props that all host activity items must adhere to */
export interface IHostActivityItemComponentProps {
  activity: IPastActivity;
  // TODO: two types, one for optional and one for required onShowDetails.
  onShowDetails?: ShowActivityDetailsHandler;
}

export const pastActivityComponentMap: Record<
  IHostPastActivityType,
  React.FC<IHostActivityItemComponentProps>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
};
