import React, { useContext } from "react";
import { AppContext } from "context/app";
import { addHours, isPast } from "date-fns";

import {
  DiskEncryptionStatus,
  MdmEnrollmentStatus,
  isAutomaticDeviceEnrollment,
} from "interfaces/mdm";
import { IOSSettings } from "interfaces/host";
import {
  HostPlatform,
  isDiskEncryptionSupportedLinuxPlatform,
} from "interfaces/platform";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import {
  INITIAL_FLEET_DATE,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

const baseClass = "host-details-banners";

export interface IHostBannersBaseProps {
  macDiskEncryptionStatus: DiskEncryptionStatus | null | undefined;
  mdmEnrollmentStatus: MdmEnrollmentStatus | null;
  connectedToFleetMdm?: boolean;
  hostPlatform?: HostPlatform;
  // used to identify Fedora hosts, whose platform is "rhel"
  hostOsVersion?: string;
  /** Disk encryption setting status and detail, if any, that apply to this host (via a team or the "no team" team) */
  diskEncryptionOSSetting?: IOSSettings["disk_encryption"];
  /** Whether or not this host's disk is encrypted */
  diskIsEncrypted?: boolean;
  /** Whether or not Fleet has escrowed the host's disk encryption key */
  diskEncryptionKeyAvailable?: boolean;
  /** The timestamp of the last MDM enrollment */
  lastMdmEnrolledAt?: string;
  /** The timestamp of the last detail update */
  detailUpdatedAt?: string;
}
/**
 * Handles the displaying of banners on the host details page
 */
const HostDetailsBanners = ({
  mdmEnrollmentStatus,
  hostPlatform,
  hostOsVersion,
  connectedToFleetMdm,
  macDiskEncryptionStatus,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
  lastMdmEnrolledAt,
  detailUpdatedAt,
}: IHostBannersBaseProps) => {
  const { config } = useContext(AppContext);

  const isMdmUnenrolled = mdmEnrollmentStatus === "Off" || !mdmEnrollmentStatus;
  const isNewMdmEnrollment =
    !isMdmUnenrolled &&
    !!lastMdmEnrolledAt &&
    // if less than an hour has passed since the last MDM enrollment, we consider it a new
    // enrollment and won't show the disk encryption action required banner, as it's possible the
    // host just hasn't sent its disk encryption status to Fleet yet
    !isPast(addHours(lastMdmEnrolledAt, 1));

  const showTurnOnMdmInfoBanner =
    hostPlatform === "darwin" &&
    isMdmUnenrolled &&
    config?.mdm.enabled_and_configured &&
    detailUpdatedAt &&
    detailUpdatedAt > INITIAL_FLEET_DATE;

  const showMacDiskEncryptionUserActionRequired =
    config?.mdm.enabled_and_configured &&
    connectedToFleetMdm &&
    macDiskEncryptionStatus === "action_required" &&
    !isNewMdmEnrollment;

  // ADE-enrolled hosts escrow their FileVault key automatically, so the end user
  // doesn't need to log out. Manually-enrolled hosts only get a new key at next
  // login, so they keep the log-out instruction.
  const isAdeEnrolled = isAutomaticDeviceEnrollment(mdmEnrollmentStatus);

  const actionRequiredBanner = (
    <div className={baseClass}>
      <InfoBanner color="yellow">
        Disk encryption: Requires action from the end user. Ask the user to
        follow <b>Disk encryption</b> instructions on their <b>My device</b>{" "}
        page.
      </InfoBanner>
    </div>
  );

  if (showTurnOnMdmInfoBanner) {
    return (
      <div className={baseClass}>
        <InfoBanner color="yellow">
          To enforce settings, OS updates, disk encryption, and more, ask the
          end user to follow the <strong>Turn on MDM</strong> instructions on
          their <strong>My device</strong> page.
        </InfoBanner>
      </div>
    );
  }
  if (showMacDiskEncryptionUserActionRequired) {
    return (
      <div className={baseClass}>
        <InfoBanner color="yellow">
          {isAdeEnrolled ? (
            <>
              Disk encryption: FileVault key will be escrowed automatically on
              this host&apos;s next refetch.
            </>
          ) : (
            <>
              Disk encryption: Requires action from the end user. Ask the end
              user to log out of their device or restart it.
            </>
          )}
        </InfoBanner>
      </div>
    );
  }
  if (
    hostPlatform &&
    isDiskEncryptionSupportedLinuxPlatform(hostPlatform, hostOsVersion ?? "") &&
    diskEncryptionOSSetting?.status
  ) {
    // setting applies to a Linux host
    if (!diskIsEncrypted) {
      // linux host not in compliance with setting
      return (
        <div className={baseClass}>
          <InfoBanner
            color="yellow"
            cta={
              <CustomLink
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/mdm-disk-encryption`}
                text="Guide"
                variant="banner-link"
                newTab
              />
            }
          >
            Disk encryption: Disk encryption is off. Currently, to turn on{" "}
            <b>full</b> disk encryption, the end user has to re-install their
            operating system.
          </InfoBanner>
        </div>
      );
    }
    if (!diskEncryptionKeyAvailable) {
      // linux host's disk is encrypted, but Fleet doesn't yet have a disk
      // encryption key escrowed (note that this state is also possible for Windows hosts, which we
      // don't show this banner for currently)
      return actionRequiredBanner;
    }
  }
  if (
    hostPlatform === "windows" &&
    diskEncryptionOSSetting?.status === "action_required"
  ) {
    // Fleet is holding the repair until the host restarts, so point the admin at the restart rather than at My device.
    if (diskEncryptionOSSetting?.action_required === "restart") {
      return (
        <div className={baseClass}>
          <InfoBanner color="yellow">
            Disk encryption: Requires a restart. Ask the user to restart their
            device so disk encryption protection can be turned back on.
          </InfoBanner>
        </div>
      );
    }
    if (diskEncryptionOSSetting?.action_required === "create_pin") {
      return actionRequiredBanner;
    }
  }

  return null;
};

export default HostDetailsBanners;
