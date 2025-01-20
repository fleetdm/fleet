import { getErrorReason } from "interfaces/errors";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const reason = getErrorReason(err);
  return reason;
};
