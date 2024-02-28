import React from "react";

import {
  ActivityType,
  IHostPastActivityType,
  IHostUpcomingActivityType,
  IPastActivity,
} from "interfaces/activity";

import { ShowActivityDetailsHandler } from "./Activity";

import RanScriptActivityItem from "./ActivityItems/RanScriptActivityItem";
import LockedHostActivityItem from "./ActivityItems/LockedHostActivityItem";
import UnlockedHostActivityItem from "./ActivityItems/UnlockedHostActivityItem";
import RanMdmCommandActivityItem from "./ActivityItems/RanMdmCommandActivityItem";
import InstalledFleetdActivityItem from "./ActivityItems/InstalledFleetdActivityItem";
import SetAccountConfigActivityItem from "./ActivityItems/SetAccountConfigActivityItem";
import CreatedOsProfileActivityItem from "./ActivityItems/CreatedOsProfileActivityItem";
import EditedOsProfileActivityItem from "./ActivityItems/EditedOsProfileActivityItem";
import DeletedOsProfileActivityItem from "./ActivityItems/DeletedOsProfileActivityItem";
import EnabledDiskEncryptionActivityItem from "./ActivityItems/EnabledDiskEncryptionActivityItem";
import EditedMacosMinVersionActivityItem from "./ActivityItems/EditedMacosMinVersionActivityItem";
import WipedHostActivityItem from "./ActivityItems/WipedHostActivityItem";
import EditedWindowsUpdatesActivityItem from "./ActivityItems/EditedWindowsUpdatesActivityItem";

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
  [ActivityType.InstalledFleetd]: InstalledFleetdActivityItem,
  [ActivityType.SetAccountConfiguration]: SetAccountConfigActivityItem,
  [ActivityType.CreatedMacOSProfile]: CreatedOsProfileActivityItem,
  [ActivityType.EditedMacOSProfile]: EditedOsProfileActivityItem,
  [ActivityType.DeletedMacOSProfile]: DeletedOsProfileActivityItem,
  [ActivityType.CreatedWindowsProfile]: CreatedOsProfileActivityItem,
  [ActivityType.EnabledDiskEncryption]: EnabledDiskEncryptionActivityItem,
  [ActivityType.DisabledDiskEncryption]: LockedHostActivityItem,
  [ActivityType.EditedMacosMinVersion]: EditedMacosMinVersionActivityItem, // TODO: check if needs to be extended to check for removal
  [ActivityType.EditedWindowsUpdates]: EditedWindowsUpdatesActivityItem,
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
  [ActivityType.WipedHost]: WipedHostActivityItem,
  [ActivityType.RanScript]: RanScriptActivityItem,
};

export const updateActivityComponentMap: Record<
  IHostUpcomingActivityType,
  | React.FC<IHostActivityItemComponentProps>
  | React.FC<IHostActivityItemComponentPropsWithShowDetails>
> = {
  [ActivityType.RanMdmCommand]: RanMdmCommandActivityItem,
  [ActivityType.InstalledFleetd]: InstalledFleetdActivityItem,
  [ActivityType.SetAccountConfiguration]: SetAccountConfigActivityItem,
  [ActivityType.CreatedMacOSProfile]: CreatedOsProfileActivityItem,
  [ActivityType.EditedMacOSProfile]: EditedOsProfileActivityItem,
  [ActivityType.DeletedMacOSProfile]: DeletedOsProfileActivityItem,
  [ActivityType.CreatedWindowsProfile]: CreatedOsProfileActivityItem,
  [ActivityType.EnabledDiskEncryption]: EnabledDiskEncryptionActivityItem,
  [ActivityType.DisabledDiskEncryption]: LockedHostActivityItem,
  [ActivityType.EditedMacosMinVersion]: EditedMacosMinVersionActivityItem, // TODO: check if needs to be extended to check for removal
  [ActivityType.EditedWindowsUpdates]: EditedWindowsUpdatesActivityItem,
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
  [ActivityType.WipedHost]: WipedHostActivityItem,
  [ActivityType.RanScript]: RanScriptActivityItem,
};
