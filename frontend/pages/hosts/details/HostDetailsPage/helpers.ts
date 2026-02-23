import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

 
export const getErrorMessage = (e: unknown, hostName: string) => {
  let errorMessage = getErrorReason(e, {
    reasonIncludes: "Host does not have MDM turned on",
  });

  if (!errorMessage) {
    errorMessage = DEFAULT_ERROR_MESSAGE;
  }

  return `Host "${hostName}" ${errorMessage}`;
};
