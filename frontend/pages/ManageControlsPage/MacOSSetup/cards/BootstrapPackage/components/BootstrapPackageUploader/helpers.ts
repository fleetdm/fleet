import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

export const UPLOAD_ERROR_MESSAGES = {
  wrongType: {
    condition: (reason: string) => reason.includes("invalid file type"),
    message: "Couldn’t upload. The file should be a package (.pkg).",
  },
  unsigned: {
    condition: (reason: string) => reason.includes("file is not"),
    message:
      "Couldn’t upload. The package must be signed. Click “Learn more” below to learn how to sign.",
  },
  default: {
    condition: () => false,
    message: "Couldn’t upload. Please try again.",
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
