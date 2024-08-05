import React, { useContext } from "react";
import { AppContext } from "context/app";

import { DiskEncryptionStatus, MdmEnrollmentStatus } from "interfaces/mdm";
import { hasLicenseExpired, willExpireWithinXDays } from "utilities/helpers";
import InfoBanner from "components/InfoBanner";

const baseClass = "host-details-banners";

interface IHostDetailsBannersProps {
  hostMdmEnrollmentStatus?: MdmEnrollmentStatus | null;
  hostPlatform?: string;
  diskEncryptionStatus: DiskEncryptionStatus | null | undefined;
  connectedToFleetMdm?: boolean;
}

/**
 * Handles the displaying of banners on the host details page
 */
const HostDetailsBanners = ({
  hostMdmEnrollmentStatus,
  hostPlatform,
  connectedToFleetMdm,
  diskEncryptionStatus,
}: IHostDetailsBannersProps) => {
  const { config, isPremiumTier, apnsExpiry, abmExpiry } = useContext(
    AppContext
  );

  // Checks to see if an app-wide banner is being shown (the ABM terms, ABM expiry,
  // or APNs expiry banner) in a parent component. App-wide banners found in parent
  // component take priority over host details page-level banners.
  const isAppleBmTermsExpired = config?.mdm?.apple_bm_terms_expired;
  const isApplePnsExpired = hasLicenseExpired(apnsExpiry || "");
  const willApplePnsExpireIn30Days = willExpireWithinXDays(
    apnsExpiry || "",
    30
  );
  const isAppleBmExpired = hasLicenseExpired(abmExpiry || "");
  const willAppleBmExpireIn30Days = willExpireWithinXDays(abmExpiry || "", 30);
  const isFleetLicenseExpired = hasLicenseExpired(
    config?.license.expiration || ""
  );

  const showingAppWideBanner =
    isPremiumTier &&
    (isAppleBmTermsExpired ||
      isApplePnsExpired ||
      willApplePnsExpireIn30Days ||
      isAppleBmExpired ||
      willAppleBmExpireIn30Days ||
      isFleetLicenseExpired);

  const isMdmUnenrolled =
    hostMdmEnrollmentStatus === "Off" || !hostMdmEnrollmentStatus;

  const showTurnOnMdmInfoBanner =
    !showingAppWideBanner &&
    hostPlatform === "darwin" &&
    isMdmUnenrolled &&
    config?.mdm.enabled_and_configured;

  const showDiskEncryptionUserActionRequired =
    !showingAppWideBanner &&
    config?.mdm.enabled_and_configured &&
    connectedToFleetMdm &&
    diskEncryptionStatus === "action_required";

  if (showTurnOnMdmInfoBanner || showDiskEncryptionUserActionRequired) {
    return (
      <div className={baseClass}>
        {showTurnOnMdmInfoBanner && (
          <InfoBanner color="yellow">
            To enforce settings, OS updates, disk encryption, and more, ask the
            end user to follow the <strong>Turn on MDM</strong> instructions on
            their <strong>My device</strong> page.
          </InfoBanner>
        )}
        {showDiskEncryptionUserActionRequired && (
          <InfoBanner color="yellow">
            Disk encryption: Requires action from the end user. Ask the end user
            to log out of their device or restart it.
          </InfoBanner>
        )}
      </div>
    );
  }
  return null;
};

export default HostDetailsBanners;
