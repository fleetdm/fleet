import React from "react";

import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";

const baseClass = "vpp-renewal-message";

interface IVppRenewalMessageProps {
  expired: boolean;
}

const VppRenewalMessage = ({ expired }: IVppRenewalMessageProps) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url="/settings/integrations/mdm/vpp"
          text="Renew VPP"
          className={`${baseClass}`}
          color="core-fleet-black"
          iconColor="core-fleet-black"
        />
      }
    >
      {expired ? (
        <>
          Your Volume Purchasing Program (VPP) content token has expired. You
          can’t add or install App Store apps. Users with the admin role in
          Fleet can renew VPP.
        </>
      ) : (
        <>
          Your Volume Purchasing Program (VPP) content token is less than 30
          days from expiration. If it expires, you won’t be able to add or
          install App Store apps. Users with the admin role in Fleet can renew
          VPP.
        </>
      )}
    </InfoBanner>
  );
};

export default VppRenewalMessage;
