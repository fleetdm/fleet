import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

const UPLOAD_ERROR_MESSAGES = {
  default: {
    message: "Couldn’t upload. Please try again.",
  },
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err.data.errors[0].reason;
  return apiReason || UPLOAD_ERROR_MESSAGES.default.message;
};
