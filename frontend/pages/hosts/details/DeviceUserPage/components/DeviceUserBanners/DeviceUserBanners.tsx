import React from "react";
import { addHours, isPast } from "date-fns";

import InfoBanner from "components/InfoBanner";
import Button from "components/buttons/Button";
import { MacDiskEncryptionActionRequired } from "interfaces/host";
import { IHostBannersBaseProps } from "pages/hosts/details/HostDetailsPage/components/HostDetailsBanners/HostDetailsBanners";
import CustomLink from "components/CustomLink";
import { isDiskEncryptionSupportedLinuxPlatform } from "interfaces/platform";
import { isAutomaticDeviceEnrollment } from "interfaces/mdm";
import { INITIAL_FLEET_DATE } from "utilities/constants";

const baseClass = "device-user-banners";

interface IDeviceUserBannersProps extends IHostBannersBaseProps {
  mdmEnabledAndConfigured: boolean;
  diskEncryptionActionRequired: MacDiskEncryptionActionRequired | null;
  mdmManualEnrolmentUrl?: string;
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
  mdmManualEnrolmentUrl,
  onClickCreatePIN,
  onClickTurnOnMdm,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
  onTriggerEscrowLinuxKey,
  lastMdmEnrolledAt,
  detailUpdatedAt,
}: IDeviceUserBannersProps) => {
  const isMdmUnenrolled =
    mdmEnrollmentStatus === "Off" || mdmEnrollmentStatus === null;

  const mdmEnabledAndConnected = mdmEnabledAndConfigured && connectedToFleetMdm;

  const showTurnOnAppleMdmBanner =
    hostPlatform === "darwin" &&
    isMdmUnenrolled &&
    mdmEnabledAndConfigured &&
    detailUpdatedAt &&
    detailUpdatedAt > INITIAL_FLEET_DATE;

  const isNewMdmEnrollment =
    !isMdmUnenrolled &&
    !!lastMdmEnrolledAt &&
    // if less than an hour has passed since the last MDM enrollment, we consider it a new
    // enrollment and won't show the disk encryption action required banner, as it's possible the
    // host just hasn't sent its disk encryption status to Fleet yet
    !isPast(addHours(lastMdmEnrolledAt, 1));

  const showMacDiskEncryptionKeyResetRequired =
    mdmEnabledAndConnected &&
    macDiskEncryptionStatus === "action_required" &&
    diskEncryptionActionRequired === "rotate_key" &&
    !isNewMdmEnrollment;

  // ADE-enrolled hosts escrow their FileVault key automatically, so there's nothing
  // for the end user to do but refetch. Manually-enrolled hosts only get a new key at
  // next login, so they keep the log-out instruction.
  const isAdeEnrolled = isAutomaticDeviceEnrollment(mdmEnrollmentStatus);

  const turnOnMdmButton = mdmManualEnrolmentUrl ? (
    <CustomLink
      url={mdmManualEnrolmentUrl}
      text="Turn on MDM"
      newTab
      variant="banner-link"
    />
  ) : (
    <Button variant="link" onClick={onClickTurnOnMdm}>
      Turn on MDM
    </Button>
  );

  const renderBanner = () => {
    if (showTurnOnAppleMdmBanner) {
      return (
        <InfoBanner color="yellow" cta={turnOnMdmButton}>
          Mobile device management (MDM) is off. MDM allows your organization to
          change settings and install software. This lets your organization keep
          your device up to date so you don&apos;t have to.
        </InfoBanner>
      );
    }

    if (showMacDiskEncryptionKeyResetRequired) {
      return (
        <InfoBanner color="yellow">
          {isAdeEnrolled ? (
            <>
              Disk encryption: Refetch to ensure data is safeguarded in case
              your device is lost or stolen. If this banner persists, contact
              your IT admin.
            </>
          ) : (
            <>
              Disk encryption: Log out of your device or restart it to safeguard
              your data in case your device is lost or stolen. After, select{" "}
              <strong>Refetch</strong> to clear this banner.
            </>
          )}
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
                variant="secondary"
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
      // Fleet is holding the repair until the host restarts, so the restart is the only thing that moves it along.
      if (diskEncryptionOSSetting?.action_required === "restart") {
        return (
          <InfoBanner color="yellow">
            Disk encryption: Restart your device to finish protecting your data.
            Your organization will turn disk encryption protection back on after
            the restart.
          </InfoBanner>
        );
      }

      // Gate on action_required naming the PIN.
      if (diskEncryptionOSSetting?.action_required === "create_pin") {
        return (
          <InfoBanner
            color="yellow"
            cta={
              <Button variant="link" onClick={onClickCreatePIN}>
                Create PIN
              </Button>
            }
          >
            Disk encryption: Create a BitLocker PIN to safeguard your data in
            case your device is lost or stolen. After, select{" "}
            <strong>Refetch</strong> to clear this banner.
          </InfoBanner>
        );
      }
    }

    return null;
  };

  const banner = renderBanner();
  return banner ? <div className={baseClass}>{banner}</div> : null;
};

export default DeviceUserBanners;
