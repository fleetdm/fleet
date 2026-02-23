import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Couldn’t renew. Please try again.";

 
export const getErrorMessage = (err: unknown) => {
  const invalidTokenReason = getErrorReason(err, {
    reasonIncludes: "Invalid token",
  });

  if (invalidTokenReason) {
    return invalidTokenReason;
  }

  return DEFAULT_ERROR_MESSAGE;
};
