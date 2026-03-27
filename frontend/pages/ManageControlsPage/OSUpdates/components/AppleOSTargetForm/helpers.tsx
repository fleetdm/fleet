import React from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err?.data?.errors?.[0]?.reason?.toLowerCase?.();

  if (apiReason?.includes("version isn't supported by apple")) {
    return (
      <>
        Couldn&apos;t update. The <b>Minimum version</b> isn&apos;t supported by
        Apple.
      </>
    );
  }

  if (apiReason?.includes("deadline isn't a valid date")) {
    return (
      <>
        Couldn&apos;t update. The <b>Deadline</b> isn&apos;t a valid date.
      </>
    );
  }

  return "Couldn’t update. Please try again.";
};
