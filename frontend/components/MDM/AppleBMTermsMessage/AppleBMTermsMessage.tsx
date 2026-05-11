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
          text="Go to AB"
          className={`${baseClass}__new-tab`}
          newTab
          variant="banner-link"
        />
      }
    >
      You can’t automatically enroll macOS, iOS, and iPadOS hosts until you
      accept the new terms and conditions for your Apple Business (AB). An AB
      administrator can accept these terms. If you have connected multiple AB
      instances, this banner will disappear once you accept the new terms and
      conditions in all of them.
    </InfoBanner>
  );
};

export default AppleBMTermsMessage;
