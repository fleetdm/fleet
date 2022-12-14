import React from "react";

import PATHS from "router/paths";
import { InjectedRouter } from "react-router";
import Button from "components/buttons/Button";

import ExternalLinkIcon from "../../../../assets/images/icon-external-link-black-12x12@2x.png";

const baseClass = "apple-bm-terms-message";

interface IAppleBMTermsMessage {
  router: InjectedRouter; // v3
}

const AppleBMTermsMessage = ({ router }: IAppleBMTermsMessage) => {
  const onClick = (): void => {
    router.push(PATHS.MANAGE_HOSTS);
  };

  return (
    <a
      href="https://business.apple.com/"
      target="_blank"
      rel="noreferrer"
      className={baseClass}
    >
      <p>
        Your organization can’t automatically enroll macOS hosts until you
        accept the new terms and conditions for Apple Business Manager (ABM). An
        ABM administrator can accept these terms. Done?{" "}
        <Button onClick={onClick} variant="text-link">
          Go to Hosts
        </Button>{" "}
        to remove this banner.
      </p>
      <span>
        Go to ABM
        <img alt="Open external link" src={ExternalLinkIcon} />
      </span>
    </a>
  );
};

export default AppleBMTermsMessage;
