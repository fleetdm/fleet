import React, { useContext } from "react";
import { AppContext } from "context/app";

import { DiskEncryptionStatus, MdmEnrollmentStatus } from "interfaces/mdm";
import { hasLicenseExpired } from "utilities/helpers";
import InfoBanner from "components/InfoBanner";
import { IOSSettings } from "interfaces/host";
import {
  HostPlatform,
  platformSupportsDiskEncryption,
} from "interfaces/platform";

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
}: IHostBannersBaseProps) => {
  const {
    config,
    isPremiumTier,
    isAppleBmExpired,
    isApplePnsExpired,
    isVppExpired,
    needsAbmTermsRenewal,
    willAppleBmExpire,
    willApplePnsExpire,
    willVppExpire,
  } = useContext(AppContext);

  // Checks to see if an app-wide banner is being shown (the ABM terms, ABM expiry,
  // or APNs expiry banner) in a parent component. App-wide banners found in parent
  // component take priority over host details page-level banners.
  const isFleetLicenseExpired = hasLicenseExpired(
    config?.license.expiration || ""
  );

  const showingAppWideBanner =
    isPremiumTier &&
    (needsAbmTermsRenewal ||
      isApplePnsExpired ||
      willApplePnsExpire ||
      isAppleBmExpired ||
      willAppleBmExpire ||
      isVppExpired ||
      willVppExpire ||
      isFleetLicenseExpired);

  const isMdmUnenrolled = mdmEnrollmentStatus === "Off" || !mdmEnrollmentStatus;

  const showTurnOnMdmInfoBanner =
    !showingAppWideBanner &&
    hostPlatform === "darwin" &&
    isMdmUnenrolled &&
    config?.mdm.enabled_and_configured;

  const showMacDiskEncryptionUserActionRequired =
    !showingAppWideBanner &&
    config?.mdm.enabled_and_configured &&
    connectedToFleetMdm &&
    macDiskEncryptionStatus === "action_required";

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
          Disk encryption: Requires action from the end user. Ask the end user
          to log out of their device or restart it.
        </InfoBanner>
      </div>
    );
  }
  // setting applies
  if (
    hostPlatform &&
    platformSupportsDiskEncryption(hostPlatform, hostOsVersion) &&
    diskEncryptionOSSetting?.status
  ) {
    // host either not in compliance with setting, or is but Fleet doesn't yet have a disk
    // encryption key escrowed for the host (possible for Linux hosts)
    if (!diskIsEncrypted || !diskEncryptionKeyAvailable) {
      return (
        <div className={baseClass}>
          <InfoBanner color="yellow">
            Disk encryption: Requires action from the end user. Ask the user to
            follow <b>Disk encryption</b> instructions on their <b>My device</b>{" "}
            page.
          </InfoBanner>
        </div>
      );
    }
  }
  return null;
};

export default HostDetailsBanners;
