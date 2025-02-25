import React from "react";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";

const baseClass = "apple-pn-cert-renewal-message";

type IApplePNCertRenewalMessage = {
  expired: boolean;
};

const ApplePNCertRenewalMessage = ({ expired }: IApplePNCertRenewalMessage) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="https://fleetdm.com/learn-more-about/renew-apns"
          text="Renew APNs"
          className={`${baseClass}__new-tab`}
          newTab
          color="core-fleet-black"
          iconColor="core-fleet-black"
        />
      }
    >
      {expired ? (
        <>
          Your Apple Push Notification service (APNs) certificate has expired.
          After you renew the certificate, all end users have to turn MDM off
          and back on.
        </>
      ) : (
        <>
          Your Apple Push Notification service (APNs) certificate is less than
          30 days from expiration. If it expires all end users will have to turn
          MDM off and back on.
        </>
      )}
    </InfoBanner>
  );
};

export default ApplePNCertRenewalMessage;
