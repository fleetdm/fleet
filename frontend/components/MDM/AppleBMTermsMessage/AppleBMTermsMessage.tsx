import React from "react";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";

const baseClass = "apple-bm-terms-message";

const AppleBMTermsMessage = () => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="https://business.apple.com/"
          text="Go to ABM"
          className={`${baseClass}__new-tab`}
          newTab
          color="core-fleet-black"
          iconColor="core-fleet-black"
        />
      }
    >
      Your organization can&apos;t automatically enroll macOS hosts until you
      accept the new terms and conditions for Apple Business Manager (ABM). An
      ABM administrator can accept these terms. Done? It might take some time
      for ABM to report back to Fleet.
    </InfoBanner>
  );
};

export default AppleBMTermsMessage;
