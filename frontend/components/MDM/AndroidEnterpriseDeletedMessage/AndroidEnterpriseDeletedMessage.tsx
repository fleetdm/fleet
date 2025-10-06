import React from "react";

import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";

const baseClass = "android-enterprise-deleted-message";

const AndroidEnterpriseDeletedMessage = () => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="https://fleetdm.com/learn-more-about/how-to-connect-android-enterprise"
          text="Learn more"
          className={baseClass}
          newTab
          variant="banner-link"
        />
      }
    >
      Android MDM is off because Android Enterprise was deleted in Google
      Console. Please reconnect Android Enterprise.
    </InfoBanner>
  );
};

export default AndroidEnterpriseDeletedMessage;
