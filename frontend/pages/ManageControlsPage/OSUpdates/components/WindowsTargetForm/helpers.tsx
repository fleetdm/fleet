import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const originalReason = err?.data?.errors?.[0]?.reason;
  const apiReason = originalReason?.toLowerCase();

  if (apiReason?.includes("couldn't update os updates settings")) {
    return originalReason;
  }

  return "Couldn’t update. Please try again.";
};
