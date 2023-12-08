import React from "react";
import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

/** We want to add some additional messageing to some of the error messages so
 * we add them in this function. Otherwise, we'll just return the error message from the
 * API.
 */
// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err.data.errors[0].reason;

  if (
    apiReason.includes(
      "The configuration profile can’t include BitLocker settings."
    )
  ) {
    return (
      <span>
        {apiReason} To control these settings, go to <b>Disk encryption</b>.
      </span>
    );
  }

  if (
    apiReason.includes(
      "The configuration profile can’t include Windows update settings."
    )
  ) {
    return (
      <span>
        {apiReason} To control these settings, go to <b>OS updates</b>.
      </span>
    );
  }
  return apiReason;
};
