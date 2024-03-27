import { getErrorReason } from "interfaces/errors";

const UPLOAD_ERROR_MESSAGES = {
  default: {
    message: "Couldn't upload. Please try again.",
  },
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  if (typeof err === "string") return err;
  return getErrorReason(err) || UPLOAD_ERROR_MESSAGES.default.message;
};
