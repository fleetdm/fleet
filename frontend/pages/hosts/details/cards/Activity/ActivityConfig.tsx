import React from "react";

import {
  ActivityType,
  IHostPastActivityType,
  IPastActivity,
} from "interfaces/activity";

import { ShowActivityDetailsHandler } from "./Activity";

import RanScriptActivityItem from "./RanScriptActivityItem";
import LockedHostActivityItem from "./LockedHostActivityItem";
import UnlockedHostActivityItem from "./UnlockedHostActivityItem";

/** the component props that all host activity items must adhere to */
export interface IHostActivityItemComponentProps {
  activity: IPastActivity;
  onShowDetails?: ShowActivityDetailsHandler;
}

// eslint-disable-next-line import/prefer-default-export
export const pastActivityComponentMap: Record<
  IHostPastActivityType,
  React.FC<IHostActivityItemComponentProps>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
};
