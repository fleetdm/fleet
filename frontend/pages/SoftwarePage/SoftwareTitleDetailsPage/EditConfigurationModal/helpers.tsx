import React from "react";

import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE =
  "Couldn't update configuration. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const reason = getErrorReason(err);

  if (
    reason.includes("managedConfiguration") ||
    reason.includes("workProfileWidgets")
  ) {
    return (
      <>
        Couldn&apos;t update configuration. Only
        &quot;managedConfiguration&quot; and &quot;workProfileWidgets&quot; are
        supported as top-level keys.
      </>
    );
  }

  return reason || DEFAULT_ERROR_MESSAGE;
};
