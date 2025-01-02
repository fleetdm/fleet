import { getErrorReason } from "interfaces/errors";

/**
 * helper function to generate error message for secret variables based
 * on the error reason.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateSecretErrMsg = (err: unknown) => {
  const reason = getErrorReason(err);

  let errorType = "";
  if (getErrorReason(err, { nameEquals: "install script" })) {
    errorType = "install script";
  } else if (getErrorReason(err, { nameEquals: "post-install script" })) {
    errorType = "post-install script";
  } else if (getErrorReason(err, { nameEquals: "uninstall script" })) {
    errorType = "uninstall script";
  } else if (getErrorReason(err, { nameEquals: "profile" })) {
    errorType = "profile";
  }

  if (errorType === "profile") {
    return reason
      .split(":")[1]
      .replace(/Secret variables?/i, "Variable")
      .replace("missing from database", "doesn't exist.");
  }

  // all other specific error types
  if (errorType) {
    return reason
      .replace(/Secret variables?/i, `Variable used in ${errorType} `)
      .replace("missing from database", "doesn't exist.");
  }

  // no spcial error type. return generic secret error message
  return reason
    .replace(/Secret variables?/i, "Variable")
    .replace("missing from database", "doesn't exist.");
};
