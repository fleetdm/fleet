import React from "react";

import PATHS from "router/paths";
import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";

const baseClass = "microsoft-graph-credential-invalid-message";

/**
 * Shown when a stored Microsoft Graph credential has been rejected by Entra. The flag is raised by the Autopilot sync
 * rather than at save time, so it can trail the actual failure by up to one sync interval, and it clears on its own
 * once a working credential is saved.
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
      Your Microsoft Graph credential is invalid. Windows Autopilot devices
      won’t sync to Fleet as pending hosts. Users with the admin role in Fleet
      can update the credential.
    </InfoBanner>
  );
};

export default MicrosoftGraphCredentialInvalidMessage;
