import React from "react";

import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";

const baseClass = "license-host-limit-message";

interface ILicenseHostLimitMessageProps {
  hostCount: number;
  licensedHostCount: number;
}

/**
 * Shown when more hosts are enrolled than the Fleet Premium license allows.
 */
const LicenseHostLimitMessage = ({
  hostCount,
  licensedHostCount,
}: ILicenseHostLimitMessageProps) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="https://fleetdm.com/learn-more-about/add-hosts-to-license"
          text="Learn how"
          newTab
          variant="banner-link"
        />
      }
    >
      You have {hostCount.toLocaleString()} hosts enrolled but are licensed for
      only {licensedHostCount.toLocaleString()}. Remove inactive hosts or add
      more to your license.
    </InfoBanner>
  );
};

export default LicenseHostLimitMessage;
