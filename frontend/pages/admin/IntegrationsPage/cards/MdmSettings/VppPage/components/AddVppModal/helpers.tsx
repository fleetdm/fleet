import React from "react";

import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Couldn’t add. Please try again.";

const generateDuplicateMessage = (msg: string) => {
  const orgName = msg.split("'")[1];
  return (
    <>
      Couldn&apos;t add. There&apos;s already a VPP connection for the{" "}
      <b>{orgName}</b> location.
    </>
  );
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const duplicateEntryReason = getErrorReason(err, {
    reasonIncludes: "Duplicate entry",
  });
  const invalidTokenReason = getErrorReason(err, {
    reasonIncludes: "Invalid token",
  });

  if (duplicateEntryReason) {
    return generateDuplicateMessage(duplicateEntryReason);
  }

  if (invalidTokenReason) {
    return invalidTokenReason;
  }

  return DEFAULT_ERROR_MESSAGE;
};
