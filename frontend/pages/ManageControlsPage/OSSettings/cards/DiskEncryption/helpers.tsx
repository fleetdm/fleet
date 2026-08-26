import React from "react";

import { getErrorReason } from "interfaces/errors";
import CustomLink from "components/CustomLink";

const PRIVATE_KEY_LEARN_MORE_LINK =
  "https://fleetdm.com/learn-more-about/fleet-server-private-key";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const reason = getErrorReason(err);

  if (reason.includes("Missing required private key")) {
    return (
      <>
        Couldn&apos;t enable disk encryption. Please configure a private key.{" "}
        <CustomLink
          url={PRIVATE_KEY_LEARN_MORE_LINK}
          text="Learn how"
          newTab
          variant="flash-message-link"
        />
      </>
    );
  }

  return (
    reason || "Could not update the disk encryption settings. Please try again."
  );
};
