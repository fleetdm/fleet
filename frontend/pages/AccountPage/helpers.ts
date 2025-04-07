import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Could not update. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  let errorMessage = getErrorReason(e, {
    reasonIncludes: "Cannot reuse old password",
  });

  if (!errorMessage) {
    errorMessage = DEFAULT_ERROR_MESSAGE;
  }

  return errorMessage;
};
