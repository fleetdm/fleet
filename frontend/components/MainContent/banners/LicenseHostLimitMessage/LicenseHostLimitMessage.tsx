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
          url="https://fleetdm.com/support"
          text="Contact Fleet"
          newTab
          variant="banner-link"
        />
      }
    >
      You&apos;re using {hostCount.toLocaleString()} of{" "}
      {licensedHostCount.toLocaleString()} Fleet Premium hosts. Remove inactive
      hosts or add more hosts to your license.
    </InfoBanner>
  );
};

export default LicenseHostLimitMessage;
