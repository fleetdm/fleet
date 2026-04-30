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
import WipedHostActivityItem from "./ActivityItems/WipedHostActivityItem";
import UnlockedHostActivityItem from "./ActivityItems/UnlockedHostActivityItem";
import ReadHostDiskEncryptionKeyActivityItem from "./ActivityItems/ReadHostDiskEncryptionKey";
import ViewedHostRecoveryLockPasswordActivityItem from "./ActivityItems/ViewedHostRecoveryLockPassword";
import SetHostRecoveryLockPasswordActivityItem from "./ActivityItems/SetHostRecoveryLockPassword";
import RotatedHostRecoveryLockPasswordActivityItem from "./ActivityItems/RotatedHostRecoveryLockPassword";
import InstalledSoftwareActivityItem from "./ActivityItems/InstalledSoftwareActivityItem";
import CanceledRunScriptActivityItem from "./ActivityItems/CanceledRunScriptActivityItem";
import CanceledInstallSoftwareActivityItem from "./ActivityItems/CanceledInstallSoftwareActivityItem";
import CanceledSetupExperienceActivityItem from "./ActivityItems/CanceledSetupExperienceActivityItem";
import CanceledUninstallSoftwareActivtyItem from "./ActivityItems/CanceledUninstallSoftwareActivtyItem";
import InstalledCertificateActivityItem from "./ActivityItems/InstalledCertificateActivityItem";
import ResentCertificateActivityItem from "./ActivityItems/ResentCertificateActivityItem";
import ClearedPasscodeActivityItem from "./ActivityItems/ClearedPasscodeActivityItem";
import FailedWipeActivityItem from "./ActivityItems/FailedWipeActivityItem";
import ViewedManagedLocalAccountActivityItem from "./ActivityItems/ViewedManagedLocalAccountActivityItem/ViewedManagedLocalAccountActivityItem";
import CreatedManagedLocalAccountActivityItem from "./ActivityItems/CreatedManagedLocalAccountActivityItem/CreatedManagedLocalAccountActivityItem";
import RotatedManagedLocalAccountPasswordActivityItem from "./ActivityItems/RotatedManagedLocalAccountPassword";

/** The component props that all host activity items must adhere to */
export interface IHostActivityItemComponentProps {
  activity: IHostPastActivity | IHostUpcomingActivity;
  tab: "past" | "upcoming";
  /** Set this to `true` when rendering only this activity by itself. This will
   * change the styles for the activity item for solo rendering.
   * @default false */
  isSoloActivity?: boolean;
  /** Set this to `true` to hide the close button and prevent from rendering
   * @default false
   */
  hideCancel?: boolean;
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
  [ActivityType.WipedHost]: WipedHostActivityItem,
  [ActivityType.FailedWipe]: FailedWipeActivityItem,
  [ActivityType.ReadHostDiskEncryptionKey]: ReadHostDiskEncryptionKeyActivityItem,
  [ActivityType.ViewedHostRecoveryLockPassword]: ViewedHostRecoveryLockPasswordActivityItem,
  [ActivityType.SetHostRecoveryLockPassword]: SetHostRecoveryLockPasswordActivityItem,
  [ActivityType.RotatedHostRecoveryLockPassword]: RotatedHostRecoveryLockPasswordActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
  [ActivityType.InstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.UninstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAppStoreApp]: InstalledSoftwareActivityItem,
  [ActivityType.CanceledRunScript]: CanceledRunScriptActivityItem,
  [ActivityType.CanceledInstallSoftware]: CanceledInstallSoftwareActivityItem,
  [ActivityType.CanceledInstallAppStoreApp]: CanceledInstallSoftwareActivityItem,
  [ActivityType.CanceledUninstallSoftware]: CanceledUninstallSoftwareActivtyItem,
  [ActivityType.CanceledSetupExperience]: CanceledSetupExperienceActivityItem,
  [ActivityType.InstalledCertificate]: InstalledCertificateActivityItem,
  [ActivityType.ResentCertificate]: ResentCertificateActivityItem,
  [ActivityType.ClearedPasscode]: ClearedPasscodeActivityItem,
  [ActivityType.ViewedManagedLocalAccount]: ViewedManagedLocalAccountActivityItem,
  [ActivityType.CreatedManagedLocalAccount]: CreatedManagedLocalAccountActivityItem,
  [ActivityType.RotatedManagedLocalAccountPassword]: RotatedManagedLocalAccountPasswordActivityItem,
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
  [ActivityType.LockedHost]: LockedHostActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
};
