import React from "react";

import InfoBanner from "components/InfoBanner";
import Button from "components/buttons/Button";
import { DiskEncryptionStatus, MdmEnrollmentStatus } from "interfaces/mdm";
import { MacDiskEncryptionActionRequired } from "interfaces/host";

const baseClass = "device-user-banners";

interface IDeviceUserBannersProps {
  hostPlatform: string;
  mdmEnrollmentStatus: MdmEnrollmentStatus | null;
  mdmEnabledAndConfigured: boolean;
  mdmConnectedToFleet: boolean;
  diskEncryptionStatus: DiskEncryptionStatus | null;
  diskEncryptionActionRequired: MacDiskEncryptionActionRequired | null;
  onTurnOnMdm: () => void;
  onResetKey: () => void;
}

const DeviceUserBanners = ({
  hostPlatform,
  mdmEnrollmentStatus,
  mdmEnabledAndConfigured,
  mdmConnectedToFleet,
  diskEncryptionStatus,
  diskEncryptionActionRequired,
  onTurnOnMdm,
  onResetKey,
}: IDeviceUserBannersProps) => {
  const isMdmUnenrolled =
    mdmEnrollmentStatus === "Off" || mdmEnrollmentStatus === null;

  const diskEncryptionBannersEnabled =
    mdmEnabledAndConfigured && mdmConnectedToFleet;

  const showTurnOnMdmBanner =
    hostPlatform === "darwin" && isMdmUnenrolled && mdmEnabledAndConfigured;

  const showDiskEncryptionLogoutRestart =
    diskEncryptionBannersEnabled &&
    diskEncryptionStatus === "action_required" &&
    diskEncryptionActionRequired === "log_out";

  const showDiskEncryptionKeyResetRequired =
    diskEncryptionBannersEnabled &&
    diskEncryptionStatus === "action_required" &&
    diskEncryptionActionRequired === "rotate_key";

  const turnOnMdmButton = (
    <Button variant="unstyled" onClick={onTurnOnMdm}>
      <b>Turn on MDM</b>
    </Button>
  );

  const resetKeyButton = (
    <Button variant="unstyled" onClick={onResetKey}>
      <b>Reset key</b>
    </Button>
  );

  const renderBanner = () => {
    if (showTurnOnMdmBanner) {
      return (
        <InfoBanner color="yellow" cta={turnOnMdmButton}>
          Mobile device management (MDM) is off. MDM allows your organization to
          change settings and install software. This lets your organization keep
          your device up to date so you don&apos;t have to.
        </InfoBanner>
      );
    } else if (showDiskEncryptionLogoutRestart) {
      return (
        <InfoBanner color="yellow">
          Disk encryption: Log out of your device or restart to turn on disk
          encryption. Then, select <strong>Refetch</strong>. This prevents
          unauthorized access to the information on your device.
        </InfoBanner>
      );
    } else if (showDiskEncryptionKeyResetRequired) {
      return (
        <InfoBanner color="yellow" cta={resetKeyButton}>
          Disk encryption: Reset your disk encryption key. This lets your
          organization help you unlock your device if you forget your password.
        </InfoBanner>
      );
    }

    return null;
  };

  return <div className={baseClass}>{renderBanner()}</div>;
};

export default DeviceUserBanners;
