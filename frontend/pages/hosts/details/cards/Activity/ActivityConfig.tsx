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
import RanMdmCommandActivityItem from "./ActivityItems/RanMdmCommandActivityItem";

/** The component props that all host activity items must adhere to */
export interface IHostActivityItemComponentProps {
  activity: IPastActivity;
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
  [ActivityType.RanMdmCommand]: RanMdmCommandActivityItem,
  [ActivityType.InstalledFleetd]: LockedHostActivityItem,
  [ActivityType.SetAccountConfiguration]: LockedHostActivityItem,
  [ActivityType.CreatedMacOSProfile]: LockedHostActivityItem,
  [ActivityType.EditedMacOSProfile]: LockedHostActivityItem,
  [ActivityType.DeletedMacOSProfile]: LockedHostActivityItem,
  [ActivityType.CreatedWindowsProfile]: LockedHostActivityItem,
  [ActivityType.EnabledDiskEncryption]: LockedHostActivityItem,
  [ActivityType.DisabledDiskEncryption]: LockedHostActivityItem,
  [ActivityType.EditedMacosMinVersion]: LockedHostActivityItem,
  [ActivityType.EditedWindowsUpdates]: LockedHostActivityItem,
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
  [ActivityType.WipedHost]: UnlockedHostActivityItem,
  [ActivityType.RanScript]: RanScriptActivityItem,
};
