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
          url="https://business.apple.com/" // TODO: maybe point to new /settings/integrations/mdm/abm
          text="Go to ABM"
          className={`${baseClass}__new-tab`}
          newTab
          color="core-fleet-black"
          iconColor="core-fleet-black"
        />
      }
    >
      You can’t automatically enroll macOS, iOS, and iPadOS hosts until you
      accept the new terms and conditions for your Apple Business Manager (ABM).
      An ABM administrator can accept these terms.
    </InfoBanner>
  );
};

export default AppleBMTermsMessage;
