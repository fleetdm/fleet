import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Couldn’t renew. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const invalidTokenReason = getErrorReason(err, {
    reasonIncludes: "invalid",
  });

  if (invalidTokenReason) {
    return "Invalid token. Please provide a valid token from Apple Business Manager.";
  }

  return DEFAULT_ERROR_MESSAGE;
};
