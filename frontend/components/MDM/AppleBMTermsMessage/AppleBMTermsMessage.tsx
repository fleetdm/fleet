import React from "react";

import PATHS from "router/paths";
import { browserHistory } from "react-router";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

const baseClass = "apple-bm-terms-message";

const AppleBMTermsMessage = () => {
  const onClick = (): void => {
    browserHistory.push(PATHS.MANAGE_HOSTS);
  };

  return (
    <div className={baseClass}>
      <p>
        Your organization can’t automatically enroll macOS hosts until you
        accept the new terms and conditions for Apple Business Manager (ABM). An
        ABM administrator can accept these terms. Done?{" "}
        <Button onClick={onClick} variant="unstyled">
          Go to Hosts
        </Button>{" "}
        to remove this banner.
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
