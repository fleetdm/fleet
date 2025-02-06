import React from "react";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";

const baseClass = "license-expiry-banner";

const LicenseExpirationBanner = (): JSX.Element => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="https://fleetdm.com/learn-more-about/downgrading"
          text="Downgrade or renew"
          newTab
          variant="banner-link"
        />
      }
    >
      Your Fleet Premium license is about to expire.
    </InfoBanner>
  );
};

export default LicenseExpirationBanner;
