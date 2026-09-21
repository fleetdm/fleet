import React from "react";

import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";
import {
  ActivityType,
  IHostPastActivityType,
  IHostPastActivity,
  IHostUpcomingActivityType,
  IHostUpcomingActivity,
} from "interfaces/activity";

import CanceledInstallSoftwareActivityItem from "./ActivityItems/CanceledInstallSoftwareActivityItem";
import CanceledMdmCommandActivityItem from "./ActivityItems/CanceledMdmCommandActivityItem";
import CanceledRunScriptActivityItem from "./ActivityItems/CanceledRunScriptActivityItem";
import CanceledSetupExperienceActivityItem from "./ActivityItems/CanceledSetupExperienceActivityItem";
import CanceledUninstallSoftwareActivtyItem from "./ActivityItems/CanceledUninstallSoftwareActivtyItem";
import ClearedPasscodeActivityItem from "./ActivityItems/ClearedPasscodeActivityItem";
import CreatedDiskEncryptionPINActivityItem from "./ActivityItems/CreatedDiskEncryptionPINActivityItem/CreatedDiskEncryptionPINActivityItem";
import CreatedManagedLocalAccountActivityItem from "./ActivityItems/CreatedManagedLocalAccountActivityItem/CreatedManagedLocalAccountActivityItem";
import EditedCustomHostVitalValueActivityItem from "./ActivityItems/EditedCustomHostVitalValueActivityItem";
import FailedEnrollmentProfileRenewalActivityItem from "./ActivityItems/FailedEnrollmentProfileRenewalActivityItem";
import FailedToRotateManagedLocalAccountPasswordActivityItem from "./ActivityItems/FailedToRotateManagedLocalAccountPassword";
import FailedWipeActivityItem from "./ActivityItems/FailedWipeActivityItem";
import InstalledAllSelfServiceSoftwareActivityItem from "./ActivityItems/InstalledAllSelfServiceSoftwareActivityItem";
import InstalledCertificateActivityItem from "./ActivityItems/InstalledCertificateActivityItem";
import InstalledSoftwareActivityItem from "./ActivityItems/InstalledSoftwareActivityItem";
import LockedHostActivityItem from "./ActivityItems/LockedHostActivityItem";
import MdmEnrolledActivityItem from "./ActivityItems/MdmEnrolledActivityItem";
import MdmUnenrolledActivityItem from "./ActivityItems/MdmUnenrolledActivityItem";
import PolicyAutomationActivityItem from "./ActivityItems/PolicyAutomationActivityItem";
import RanCustomMdmCommandActivityItem from "./ActivityItems/RanCustomMdmCommandActivityItem";
import RanScriptActivityItem from "./ActivityItems/RanScriptActivityItem";
import ReadHostDiskEncryptionKeyActivityItem from "./ActivityItems/ReadHostDiskEncryptionKey";
import ReleasedFromABActivityItem from "./ActivityItems/ReleasedFromABActivityItem";
import ResentCertificateActivityItem from "./ActivityItems/ResentCertificateActivityItem";
import ResentConfigurationProfileActivityItem from "./ActivityItems/ResentConfigurationProfileActivityItem/ResentConfigurationProfileActivityItem";
import ResetPolicyActivityItem from "./ActivityItems/ResetPolicyActivityItem";
import RetrievedHostMyDeviceURLActivityItem from "./ActivityItems/RetrievedHostMyDeviceURLActivityItem";
import RotatedHostRecoveryLockPasswordActivityItem from "./ActivityItems/RotatedHostRecoveryLockPassword";
import RotatedManagedLocalAccountPasswordActivityItem from "./ActivityItems/RotatedManagedLocalAccountPassword";
import SetHostRecoveryLockPasswordActivityItem from "./ActivityItems/SetHostRecoveryLockPassword";
import UnlockedHostActivityItem from "./ActivityItems/UnlockedHostActivityItem";
import ViewedHostRecoveryLockPasswordActivityItem from "./ActivityItems/ViewedHostRecoveryLockPassword";
import ViewedManagedLocalAccountActivityItem from "./ActivityItems/ViewedManagedLocalAccountActivityItem/ViewedManagedLocalAccountActivityItem";
import WipedHostActivityItem from "./ActivityItems/WipedHostActivityItem";

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
  [ActivityType.RetrievedHostMyDeviceURL]: RetrievedHostMyDeviceURLActivityItem,
  [ActivityType.ViewedHostRecoveryLockPassword]: ViewedHostRecoveryLockPasswordActivityItem,
  [ActivityType.SetHostRecoveryLockPassword]: SetHostRecoveryLockPasswordActivityItem,
  [ActivityType.RotatedHostRecoveryLockPassword]: RotatedHostRecoveryLockPasswordActivityItem,
  [ActivityType.UnlockedHost]: UnlockedHostActivityItem,
  [ActivityType.InstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAllSelfServiceSoftware]: InstalledAllSelfServiceSoftwareActivityItem,
  [ActivityType.UninstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAppStoreApp]: InstalledSoftwareActivityItem,
  [ActivityType.CanceledRunScript]: CanceledRunScriptActivityItem,
  [ActivityType.CanceledMdmCommand]: CanceledMdmCommandActivityItem,
  [ActivityType.CanceledInstallSoftware]: CanceledInstallSoftwareActivityItem,
  [ActivityType.CanceledInstallAppStoreApp]: CanceledInstallSoftwareActivityItem,
  [ActivityType.CanceledUninstallSoftware]: CanceledUninstallSoftwareActivtyItem,
  [ActivityType.CanceledSetupExperience]: CanceledSetupExperienceActivityItem,
  [ActivityType.InstalledCertificate]: InstalledCertificateActivityItem,
  [ActivityType.ResentCertificate]: ResentCertificateActivityItem,
  [ActivityType.ClearedPasscode]: ClearedPasscodeActivityItem,
  [ActivityType.ViewedManagedLocalAccount]: ViewedManagedLocalAccountActivityItem,
  [ActivityType.CreatedManagedLocalAccount]: CreatedManagedLocalAccountActivityItem,
  [ActivityType.CreatedDiskEncryptionPIN]: CreatedDiskEncryptionPINActivityItem,
  [ActivityType.RotatedManagedLocalAccountPassword]: RotatedManagedLocalAccountPasswordActivityItem,
  [ActivityType.FailedToRotateManagedLocalAccountPassword]: FailedToRotateManagedLocalAccountPasswordActivityItem,
  [ActivityType.FailedEnrollmentProfileRenewal]: FailedEnrollmentProfileRenewalActivityItem,
  [ActivityType.MdmUnenrolled]: MdmUnenrolledActivityItem,
  [ActivityType.MdmEnrolled]: MdmEnrolledActivityItem,
  [ActivityType.RanCustomMdmCommand]: RanCustomMdmCommandActivityItem,
  [ActivityType.EditedCustomHostVitalValue]: EditedCustomHostVitalValueActivityItem,
  [ActivityType.RanAutomationWebhook]: PolicyAutomationActivityItem,
  [ActivityType.RanAutomationTicket]: PolicyAutomationActivityItem,
  [ActivityType.RanAutomationCalendarEvent]: PolicyAutomationActivityItem,
  [ActivityType.RanAutomationConditionalAccess]: PolicyAutomationActivityItem,
  [ActivityType.FailedAutomationWebhook]: PolicyAutomationActivityItem,
  [ActivityType.FailedAutomationTicket]: PolicyAutomationActivityItem,
  [ActivityType.FailedAutomationCalendarEvent]: PolicyAutomationActivityItem,
  [ActivityType.FailedAutomationConditionalAccess]: PolicyAutomationActivityItem,
  [ActivityType.ReleasedDeviceFromAB]: ReleasedFromABActivityItem,
  [ActivityType.ResentConfigurationProfile]: ResentConfigurationProfileActivityItem,
  [ActivityType.ResetPolicy]: ResetPolicyActivityItem,
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
