import React from "react";

import { isDiskEncryptionSupportedLinuxPlatform } from "interfaces/platform";
import { MacDiskEncryptionActionRequired } from "interfaces/host";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import Button from "components/buttons/Button";

import { IHostBannersBaseProps } from "pages/hosts/details/HostDetailsPage/components/HostDetailsBanners/HostDetailsBanners";

const baseClass = "device-user-banners";

interface IDeviceUserBannersProps extends IHostBannersBaseProps {
  mdmEnabledAndConfigured: boolean;
  diskEncryptionActionRequired: MacDiskEncryptionActionRequired | null;
  deviceAssignedToFleetABM?: boolean;
  onClickCreatePIN: () => void;
  onClickTurnOnMdm: () => void;
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
  onClickCreatePIN,
  onClickTurnOnMdm,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
  deviceAssignedToFleetABM = false,
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

  const turnOnMdmCTA = (
    <Button variant="text-link-dark" onClick={onClickTurnOnMdm}>
      <span>Turn on MDM</span>
      {!deviceAssignedToFleetABM && (
        <Icon
          name="external-link"
          className={`${baseClass}__external-link-icon`}
        />
      )}
    </Button>
  );

  const renderBanner = () => {
    if (showTurnOnAppleMdmBanner) {
      return (
        <InfoBanner color="yellow" cta={turnOnMdmCTA}>
          Mobile device management (MDM) is off. MDM allows your organization to
          change settings and install software. This lets your organization keep
          your device up to date so you don&apos;t have to.
        </InfoBanner>
      );
    }

    if (showMacDiskEncryptionKeyResetRequired) {
      return (
        <InfoBanner color="yellow">
          Disk encryption: Log out of your device or restart it to safeguard
          your data in case your device is lost or stolen. After, select{" "}
          <strong>Refetch</strong> to clear this banner.
        </InfoBanner>
      );
    }

    // setting applies to a supported Linux host
    if (
      hostPlatform &&
      isDiskEncryptionSupportedLinuxPlatform(
        hostPlatform,
        hostOsVersion ?? ""
      ) &&
      diskEncryptionOSSetting?.status
    ) {
      // host not in compliance with setting
      if (!diskIsEncrypted) {
        // banner 1
        return (
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
        );
      }
      // host disk is encrypted, so in compliance with the setting
      if (!diskEncryptionKeyAvailable) {
        // key is not escrowed: banner 3
        return (
          <InfoBanner
            cta={
              <Button
                variant="inverse"
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
        );
      }
    }

    if (
      hostPlatform === "windows" &&
      diskEncryptionOSSetting?.status === "action_required"
    ) {
      return (
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
      );
    }

    return null;
  };

  const banner = renderBanner();
  return banner ? <div className={baseClass}>{banner}</div> : null;
};

export default DeviceUserBanners;
