import React from "react";

import {
  ActivityType,
  IHostPastActivityType,
  IHostPastActivity,
  IHostUpcomingActivityType,
  IHostUpcomingActivity,
} from "interfaces/activity";

import { ShowActivityDetailsHandler } from "./Activity";

import RanScriptActivityItem from "./ActivityItems/RanScriptActivityItem";
import LockedHostActivityItem from "./ActivityItems/LockedHostActivityItem";
import UnlockedHostActivityItem from "./ActivityItems/UnlockedHostActivityItem";
import InstalledSoftwareActivityItem from "./ActivityItems/InstalledSoftwareActivityItem";

/** The component props that all host activity items must adhere to */
export interface IHostActivityItemComponentProps {
  activity: IHostPastActivity | IHostUpcomingActivity;
  tab: "past" | "upcoming";
}

/** Used for activity items component that need a show details handler */
export interface IHostActivityItemComponentPropsWithShowDetails
  extends IHostActivityItemComponentProps {
  onShowDetails: ShowActivityDetailsHandler;
}

export const pastActivityComponentMap: Record<
  IHostPastActivityType,
  | React.FC<IHostActivityItemComponentProps>
  | React.FC<IHostActivityItemComponentPropsWithShowDetails>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
  [ActivityType.InstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.UninstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAppStoreApp]: InstalledSoftwareActivityItem,
};

export const upcomingActivityComponentMap: Record<
  IHostUpcomingActivityType,
  | React.FC<IHostActivityItemComponentProps>
  | React.FC<IHostActivityItemComponentPropsWithShowDetails>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.InstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.UninstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAppStoreApp]: InstalledSoftwareActivityItem,
};
