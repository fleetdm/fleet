import React from "react";

import CustomLink from "components/CustomLink";

const baseClass = "apple-bm-terms-message";

const AppleBMTermsMessage = () => {
  return (
    <div className={baseClass}>
      <p>
        Your organization can&apos;t automatically enroll macOS hosts until you
        accept the new terms and conditions for Apple Business Manager (ABM). An
        ABM administrator can accept these terms. Done? It might take some time
        for ABM to report back to Fleet.
      </p>
      <CustomLink
        url="https://business.apple.com/"
        text="Go to ABM"
        className={`${baseClass}__new-tab`}
        newTab
        iconColor="core-fleet-black"
      />
    </div>
  );
};

export default AppleBMTermsMessage;
