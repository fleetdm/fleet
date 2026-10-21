import React from "react";

import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";
import { pluralize } from "utilities/strings/stringUtils";

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
      Your license covers {licensedHostCount.toLocaleString()}{" "}
      {pluralize(licensedHostCount, "host")}, but you have{" "}
      {hostCount.toLocaleString()} enrolled. Add hosts to your license or delete
      hosts.
    </InfoBanner>
  );
};

export default LicenseHostLimitMessage;
