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
          url="https://business.apple.com/" // TODO: get this link
          text="Learn more"
          className={baseClass}
          newTab
          variant="banner-link"
        />
      }
    >
      Android MDM is off because Android Enterprise was deleted in Google
      Workspace. Please turn on Android MDM again.
    </InfoBanner>
  );
};

export default AndroidEnterpriseDeletedMessage;
