import React from "react";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

/** Toast copy for a failed conditional access request, given the server's reason. */
// eslint-disable-next-line import/prefer-default-export
export const getConditionalAccessErrorMessage = (
  action: string,
  reason: string
): React.ReactNode => {
  if (reason.toLowerCase().includes("private key is not configured")) {
    return (
      <>
        Couldn&apos;t {action}. Please configure a private key.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/fleet-server-private-key`}
          text="Learn how"
          newTab
          variant="flash-message-link"
        />
      </>
    );
  }
  return reason
    ? `Couldn't ${action}. ${reason}`
    : "Something's gone wrong. Please try again.";
};
