import { getErrorReason } from "interfaces/errors";

const UPLOAD_ERROR_MESSAGES = {
  default: {
    message: "Couldn't add. Please try again.",
  },
};

 
export const getErrorMessage = (err: unknown) => {
  if (typeof err === "string") return err;
  return getErrorReason(err) || UPLOAD_ERROR_MESSAGES.default.message;
};
