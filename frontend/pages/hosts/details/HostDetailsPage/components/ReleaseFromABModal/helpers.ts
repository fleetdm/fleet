import { ReactNode } from "react";
import { generateGenericLearnMoreErrMsg } from "utilities/helpers";

const getErrorMessage = (backendError: string): ReactNode | string => {
  if (!backendError) {
    return "An unknown error occurred.";
  }

  const originalError = backendError;

  // By default we prefix the backend error message with "Couldn't send request"
  backendError = `Couldn't send request: ${backendError}`;

  if (originalError.includes("Error releasing device")) {
    return "Couldn't send request to release host from Apple Business. Please try again.";
  }

  if (originalError.includes("Apple rejected this request")) {
    return generateGenericLearnMoreErrMsg(originalError);
  }

  return backendError;
};

export default getErrorMessage;
