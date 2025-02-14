import React from "react";

import { getErrorReason } from "interfaces/errors";
import { FLEET_GUIDES_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";

const DEFAULT_ERROR_MESSAGE = "Unable to update Windows MDM. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  let message = getErrorReason(err, {
    nameEquals: "mdm.windows_enabled_and_configured",
  });
  if (message) {
    return (
      <>
        Couldn&apos;t turn on Windows MDM. Please configure Fleet with a
        certificate and key pair first.{" "}
        <CustomLink
          url={`${FLEET_GUIDES_BASE_LINK}/windows-mdm-setup#step-1-generate-your-certificate-and-key`}
          text="Learn more"
          newTab
          variant="flash-message-link"
        />
      </>
    );
  }

  message = getErrorReason(err, {
    nameEquals: "mdm.windows_migration_enabled",
  });
  if (message) {
    return message;
  }

  return DEFAULT_ERROR_MESSAGE;
};
