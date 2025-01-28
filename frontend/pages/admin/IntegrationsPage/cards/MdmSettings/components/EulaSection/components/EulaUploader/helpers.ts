import { noop } from "lodash";
import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

export const UPLOAD_ERROR_MESSAGES = {
  wrongType: {
    condition: (reason: string) => reason.includes("invalid file type"),
    message: "Couldn’t upload EULA. The file must be a PDF (.pdf).",
  },
  default: {
    condition: noop,
    message: "Couldn’t upload EULA. Please try again.",
  },
};

export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err.data.errors[0].reason;

  const error = Object.values(UPLOAD_ERROR_MESSAGES).find((errType) =>
    errType.condition(apiReason)
  );

  if (!error) {
    return UPLOAD_ERROR_MESSAGES.default.message;
  }

  return error.message;
};
