import React from "react";

import InfoBanner from "components/InfoBanner";
import Button from "components/buttons/Button";
import { MacDiskEncryptionActionRequired } from "interfaces/host";
import { IHostBannersBaseProps } from "pages/hosts/details/HostDetailsPage/components/HostDetailsBanners/HostDetailsBanners";
import CustomLink from "components/CustomLink";
import { isDiskEncryptionSupportedLinuxPlatform } from "interfaces/platform";

const baseClass = "device-user-banners";

interface IDeviceUserBannersProps extends IHostBannersBaseProps {
  mdmEnabledAndConfigured: boolean;
  diskEncryptionActionRequired: MacDiskEncryptionActionRequired | null;
  onTurnOnMdm: () => void;
  onClickCreatePIN: () => void;
  onTriggerEscrowLinuxKey: () => void;
}

const DeviceUserBanners = ({
  hostPlatform,
  hostOsVersion,
  mdmEnrollmentStatus,
  mdmEnabledAndConfigured,
  connectedToFleetMdm,
  macDiskEncryptionStatus,
  diskEncryptionActionRequired,
  onTurnOnMdm,
  onClickCreatePIN,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
  onTriggerEscrowLinuxKey,
}: IDeviceUserBannersProps) => {
  const isMdmUnenrolled =
    mdmEnrollmentStatus === "Off" || mdmEnrollmentStatus === null;

  const mdmEnabledAndConnected = mdmEnabledAndConfigured && connectedToFleetMdm;

  const showTurnOnAppleMdmBanner =
    hostPlatform === "darwin" && isMdmUnenrolled && mdmEnabledAndConfigured;

  const showMacDiskEncryptionKeyResetRequired =
    mdmEnabledAndConnected &&
    macDiskEncryptionStatus === "action_required" &&
    diskEncryptionActionRequired === "rotate_key";

  const turnOnMdmButton = (
    <Button variant="text-link-dark" onClick={onTurnOnMdm}>
      Turn on MDM
    </Button>
  );

  if (showTurnOnAppleMdmBanner) {
    return (
      <div className={baseClass}>
        <InfoBanner color="yellow" cta={turnOnMdmButton}>
          Mobile device management (MDM) is off. MDM allows your organization to
          change settings and install software. This lets your organization keep
          your device up to date so you don&apos;t have to.
        </InfoBanner>
      </div>
    );
  }

  if (showMacDiskEncryptionKeyResetRequired) {
    return (
      <div className={baseClass}>
        <InfoBanner color="yellow">
          Disk encryption: Log out of your device or restart it to safeguard
          your data in case your device is lost or stolen. After, select{" "}
          <strong>Refetch</strong> to clear this banner.
        </InfoBanner>
      </div>
    );
  }

  // setting applies to a supported Linux host
  if (
    hostPlatform &&
    isDiskEncryptionSupportedLinuxPlatform(hostPlatform, hostOsVersion ?? "") &&
    diskEncryptionOSSetting?.status
  ) {
    // host not in compliance with setting
    if (!diskIsEncrypted) {
      // banner 1
      return (
        <div className={baseClass}>
          <InfoBanner
            cta={
              <CustomLink
                url="https://fleetdm.com/learn-more-about/encrypt-linux-device"
                text="Guide"
                variant="banner-link"
              />
            }
            color="yellow"
          >
            Disk encryption: Follow the instructions in the guide to encrypt
            your device. This lets your organization help you unlock your device
            if you forget your password.
          </InfoBanner>
        </div>
      );
    }
    // host disk is encrypted, so in compliance with the setting
    if (!diskEncryptionKeyAvailable) {
      // key is not escrowed: banner 3
      return (
        <div className={baseClass}>
          <InfoBanner
            cta={
              <Button
                variant="text-link"
                onClick={onTriggerEscrowLinuxKey}
                className="create-key-button"
              >
                Create key
              </Button>
            }
            color="yellow"
          >
            Disk encryption: Create a new disk encryption key. This lets your
            organization help you unlock your device if you forget your
            passphrase.
          </InfoBanner>
        </div>
      );
    }
  }

  if (
    hostPlatform === "windows" &&
    diskEncryptionOSSetting?.status === "action_required"
  ) {
    return (
      <div className={baseClass}>
        <InfoBanner
          color="yellow"
          cta={
            <Button variant="text-link-dark" onClick={onClickCreatePIN}>
              Create PIN
            </Button>
          }
        >
          Disk encryption: Create a BitLocker PIN to safeguard your data in case
          your device is lost or stolen. After, select <strong>Refetch</strong>{" "}
          to clear this banner.
        </InfoBanner>
      </div>
    );
  }

  return null;
};

export default DeviceUserBanners;
