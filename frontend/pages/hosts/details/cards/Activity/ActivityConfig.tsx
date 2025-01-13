import React from "react";

import {
  ActivityType,
  IHostPastActivityType,
  IHostPastActivity,
  IHostUpcomingActivityType,
  IHostUpcomingActivity,
} from "interfaces/activity";

import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import RanScriptActivityItem from "./ActivityItems/RanScriptActivityItem";
import LockedHostActivityItem from "./ActivityItems/LockedHostActivityItem";
import UnlockedHostActivityItem from "./ActivityItems/UnlockedHostActivityItem";
import InstalledSoftwareActivityItem from "./ActivityItems/InstalledSoftwareActivityItem";
import CanceledScriptActivityItem from "./ActivityItems/CanceledScriptActivityItem";
import CanceledSoftwareInstallActivityItem from "./ActivityItems/CanceledSoftwareInstallActivityItem";

/** The component props that all host activity items must adhere to */
export interface IHostActivityItemComponentProps {
  activity: IHostPastActivity | IHostUpcomingActivity;
  tab: "past" | "upcoming";
  /** Set this to `true` when rendering only this activity by itself. This will
   * change the styles for the activity item for solo rendering.
   * @default false */
  soloActivity?: boolean;
  /** Set this to `true` to hide the close button and prevent from rendering
   * @default false
   */
  hideClose?: boolean;
}

/** Used for activity items component that need a show details handler */
export interface IHostActivityItemComponentPropsWithShowDetails
  extends IHostActivityItemComponentProps {
  onShowDetails: ShowActivityDetailsHandler;
  onCancel?: () => void;
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
  [ActivityType.CanceledScript]: CanceledScriptActivityItem,
  [ActivityType.CanceledSoftwareInstall]: CanceledSoftwareInstallActivityItem,
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
