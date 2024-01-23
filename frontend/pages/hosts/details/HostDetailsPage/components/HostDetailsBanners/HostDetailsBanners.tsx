import InfoBanner from "components/InfoBanner";
import { AppContext } from "context/app";
import { DiskEncryptionStatus, MdmEnrollmentStatus } from "interfaces/mdm";
import React, { useContext } from "react";

const baseClass = "host-details-banners";

interface IHostDetailsBannersProps {
  hostMdmEnrollmentStatus?: MdmEnrollmentStatus | null;
  hostPlatform?: string;
  mdmName?: string;
  diskEncryptionStatus: DiskEncryptionStatus | null | undefined;
}

/**
 * Handles the displaying of banners on the host details page
 */
const HostDetailsBanners = ({
  hostMdmEnrollmentStatus,
  hostPlatform,
  mdmName,
  diskEncryptionStatus,
}: IHostDetailsBannersProps) => {
  const { config, isSandboxMode, isPremiumTier } = useContext(AppContext);

  // checks to see if the ABM message is being shown in a parent component.
  // We want this to be the only banner shown to the user so we need to know
  // if it's already being shown so we can suppress other banners.
  const isAppleBmTermsExpired = config?.mdm?.apple_bm_terms_expired;
  const showingAppleABMBanner =
    (isAppleBmTermsExpired && isPremiumTier && !isSandboxMode) ?? false;

  const isMdmUnenrolled =
    hostMdmEnrollmentStatus === "Off" || !hostMdmEnrollmentStatus;

  const showTurnOnMdmInfoBanner =
    !showingAppleABMBanner &&
    hostPlatform === "darwin" &&
    isMdmUnenrolled &&
    config?.mdm.enabled_and_configured;

  const showDiskEncryptionUserActionRequired =
    !showingAppleABMBanner &&
    config?.mdm.enabled_and_configured &&
    mdmName === "Fleet" &&
    diskEncryptionStatus === "action_required";

  if (showTurnOnMdmInfoBanner || showDiskEncryptionUserActionRequired) {
    return (
      <div className={baseClass}>
        {showTurnOnMdmInfoBanner && (
          <InfoBanner color="yellow">
            To change settings and install software, ask the end user to follow
            the <strong>Turn on MDM</strong> instructions on their{" "}
            <strong>My device</strong> page.
          </InfoBanner>
        )}
        {showDiskEncryptionUserActionRequired && (
          <InfoBanner color="yellow">
            Disk encryption: Requires action from the end user. Ask the end user
            to follow <b>Disk encryption</b> instructions on their{" "}
            <b>My device</b> page.
          </InfoBanner>
        )}
      </div>
    );
  }
  return null;
};

export default HostDetailsBanners;
