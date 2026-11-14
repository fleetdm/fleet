import React from "react";

import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";
import PATHS from "router/paths";

const baseClass = "ejbca-cert-renewal-message";

interface IEJBCACertRenewalMessageProps {
  expired: boolean;
}

const EJBCACertRenewalMessage = ({
  expired,
}: IEJBCACertRenewalMessageProps) => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES}
          text="Update EJBCA"
          className={`${baseClass}`}
          variant="banner-link"
        />
      }
    >
      {expired ? (
        <>
          The client certificate of an EJBCA certificate authority (CA) has
          expired. Fleet can’t get certificates from EJBCA until you upload a
          new one. Users with the admin role in Fleet can update it.
        </>
      ) : (
        <>
          The client certificate of an EJBCA certificate authority (CA) is less
          than 30 days from expiration. If it expires, Fleet won’t be able to
          get certificates from EJBCA. Users with the admin role in Fleet can
          update it.
        </>
      )}
    </InfoBanner>
  );
};

export default EJBCACertRenewalMessage;
