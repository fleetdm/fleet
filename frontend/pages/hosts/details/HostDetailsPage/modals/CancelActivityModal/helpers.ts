import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "An error occurred. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const reason = getErrorReason(err);

  return `Couldn't cancel activity. ${reason || DEFAULT_ERROR_MESSAGE}`;
};
