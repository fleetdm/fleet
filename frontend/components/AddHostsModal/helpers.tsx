import React from "react";

import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

// eslint-disable-next-line import/prefer-default-export
export const renderAppleManualEnrollmentDisabled = (type: string) => {
  return (
    <EmptyState
      header="Manual enrollment is disabled"
      width="small"
      info={`${type} enrollment is only available through Apple Business (ADE).`}
      primaryButton={
        <CustomLink
          text="Learn more"
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-abm`}
          variant="button"
        />
      }
    />
  );
};

interface IAddHostsAvailability {
  useOneTimeEnrollSecrets: boolean;
  isLoadingSecrets: boolean;
  hasEnrollSecret: boolean;
}

/**
 * With one-time enroll secrets on, Apple MDM hosts enroll without a shared
 * secret, so the Add hosts flow has nothing to offer until a shared secret
 * exists. Loading counts as available so the entry point doesn't flash away
 * while secrets are still being fetched.
 */
export const isAddHostsAvailable = ({
  useOneTimeEnrollSecrets,
  isLoadingSecrets,
  hasEnrollSecret,
}: IAddHostsAvailability) =>
  !(useOneTimeEnrollSecrets && !isLoadingSecrets && !hasEnrollSecret);
