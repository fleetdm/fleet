import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Couldn’t renew. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const invalidTokenReason = getErrorReason(err, {
    reasonIncludes: "Invalid token",
  });

  if (invalidTokenReason) {
    return invalidTokenReason;
  }

  return DEFAULT_ERROR_MESSAGE;
};
