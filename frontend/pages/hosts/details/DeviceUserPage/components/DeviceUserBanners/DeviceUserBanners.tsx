import React from "react";

import InfoBanner from "components/InfoBanner";
import Button from "components/buttons/Button";
import { MacDiskEncryptionActionRequired } from "interfaces/host";
import { IHostBannersBaseProps } from "pages/hosts/details/HostDetailsPage/components/HostDetailsBanners/HostDetailsBanners";
import CustomLink from "components/CustomLink";

const baseClass = "device-user-banners";

interface IDeviceUserBannersProps extends IHostBannersBaseProps {
  mdmEnabledAndConfigured: boolean;
  diskEncryptionActionRequired: MacDiskEncryptionActionRequired | null;
  onTurnOnMdm: () => void;
  onTriggerEscrowLinuxKey: () => void;
}

const DeviceUserBanners = ({
  hostPlatform,
  mdmEnrollmentStatus,
  mdmEnabledAndConfigured,
  connectedToFleetMdm,
  macDiskEncryptionStatus,
  diskEncryptionActionRequired,
  onTurnOnMdm,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
  onTriggerEscrowLinuxKey,
}: IDeviceUserBannersProps) => {
  const isMdmUnenrolled =
    mdmEnrollmentStatus === "Off" || mdmEnrollmentStatus === null;

  const diskEncryptionBannersEnabled =
    mdmEnabledAndConfigured && connectedToFleetMdm;

  const showTurnOnMdmBanner =
    hostPlatform === "darwin" && isMdmUnenrolled && mdmEnabledAndConfigured;

  const showDiskEncryptionKeyResetRequired =
    diskEncryptionBannersEnabled &&
    macDiskEncryptionStatus === "action_required" &&
    diskEncryptionActionRequired === "rotate_key";

  const turnOnMdmButton = (
    <Button variant="unstyled" onClick={onTurnOnMdm}>
      <b>Turn on MDM</b>
    </Button>
  );

  const renderBanner = () => {
    // TODO - undo
    return (
      <InfoBanner
        cta={
          <Button
            variant="unstyled"
            onClick={onTriggerEscrowLinuxKey}
            className="create-key-button"
          >
            Create key
          </Button>
        }
        color="yellow"
      >
        Disk encryption: Create a new disk encryption key. This lets your
        organization help you unlock your device if you forget your passphrase.
      </InfoBanner>
    );

    if (showTurnOnMdmBanner) {
      return (
        <InfoBanner color="yellow" cta={turnOnMdmButton}>
          Mobile device management (MDM) is off. MDM allows your organization to
          enforce settings, OS updates, disk encryption, and more. This lets
          your organization keep your device up to date so you don&apos;t have
          to.
        </InfoBanner>
      );
    }

    if (showDiskEncryptionKeyResetRequired) {
      return (
        <InfoBanner color="yellow">
          Disk encryption: Log out of your device or restart it to safeguard
          your data in case your device is lost or stolen. After, select{" "}
          <strong>Refetch</strong> to clear this banner.
        </InfoBanner>
      );
    }

    // TODO - should these banners only be shown for linux?
    // setting applies
    if (diskEncryptionOSSetting?.status) {
      // host not in compliance with setting
      if (!diskIsEncrypted) {
        // banner 1
        return (
          <InfoBanner
            cta={
              <CustomLink
                url="https://fleetdm.com/learn-more-about/encrypt-linux-device"
                text="Guide"
                color="core-fleet-black"
                iconColor="core-fleet-black"
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
                variant="unstyled"
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

    return null;
  };

  return <div className={baseClass}>{renderBanner()}</div>;
};

export default DeviceUserBanners;
