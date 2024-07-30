import React from "react";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";

const baseClass = "apple-bm-renewal-message";

type IAppleBMRenewalMessageProps = {
  expired: boolean;
};

const AppleBMRenewalMessage = ({ expired }: IAppleBMRenewalMessageProps) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="https://fleetdm.com/learn-more-about/renew-abm"
          text="Renew ABM"
          className={`${baseClass}__new-tab`}
          newTab
          color="core-fleet-black"
          iconColor="core-fleet-black"
        />
      }
    >
      {expired ? (
        <>
          Your Apple Business Manager (ABM) server token has expired or is
          invalid. New macOS hosts will not automatically enroll to Fleet.
        </>
      ) : (
        <>
          Your Apple Business Manager (ABM) server token is less than 30 days
          from expiration. If it expires, new macOS hosts will not automatically
          enroll to Fleet.
        </>
      )}
    </InfoBanner>
  );
};

export default AppleBMRenewalMessage;
