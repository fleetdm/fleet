import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Couldn't upload. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const apiErrMessage = getErrorReason(err);

  if (
    apiErrMessage.includes(
      "File type not supported. Only .sh and .ps1 file type is allowed"
    )
  ) {
    return "Couldn't upload. The file should be .sh or .ps1 file.";
  } else if (apiErrMessage.includes("Secret variable")) {
    return apiErrMessage.replace("missing from database", "doesn't exist");
  } else if (apiErrMessage) {
    return apiErrMessage;
  }

  return DEFAULT_ERROR_MESSAGE;
};
