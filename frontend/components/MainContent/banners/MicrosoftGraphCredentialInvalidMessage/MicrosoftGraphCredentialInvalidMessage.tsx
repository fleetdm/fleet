import React from "react";

import PATHS from "router/paths";
import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";

const baseClass = "microsoft-graph-credential-invalid-message";

/**
 * Shown when a stored Microsoft Graph credential has been rejected by Entra.
 */
const MicrosoftGraphCredentialInvalidMessage = () => {
  return (
    <InfoBanner
      className={baseClass}
      color="yellow"
      cta={
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_MICROSOFT_GRAPH}
          text="Update credential"
          className={`${baseClass}`}
          variant="banner-link"
        />
      }
    >
      Your Microsoft Graph client secret is expired, deleted, or has missing
      permissions. Windows Autopilot devices won&apos;t sync to Fleet as pending
      hosts. Users with the admin role in Fleet can update the credential.
    </InfoBanner>
  );
};

export default MicrosoftGraphCredentialInvalidMessage;
