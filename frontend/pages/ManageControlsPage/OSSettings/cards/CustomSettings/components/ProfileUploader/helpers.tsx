import React from "react";
import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";
import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";

export const parseFile = async (file: File): Promise<[string, string]> => {
  // get the file name and extension
  const nameParts = file.name.split(".");
  const name = nameParts.slice(0, -1).join(".");
  const ext = nameParts.slice(-1)[0];

  switch (ext) {
    case "xml": {
      return [name, "Windows"];
    }
    case "mobileconfig": {
      return [name, "macOS, iOS, iPadOS"];
    }
    case "json": {
      return [name, "macOS, iOS, iPadOS"];
    }
    default: {
      throw new Error(`Invalid file type: ${ext}`);
    }
  }
};

export const DEFAULT_ERROR_MESSAGE =
  "Couldn’t add configuration profile. Please try again.";

/** We want to add some additional messageing to some of the error messages so
 * we add them in this function. Otherwise, we'll just return the error message from the
 * API.
 */
// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err?.data?.errors?.[0]?.reason;

  if (
    apiReason.includes(
      "The configuration profile can't include BitLocker settings."
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
      "The configuration profile can't include Windows update settings."
    )
  ) {
    return (
      <span>
        {apiReason} To control these settings, go to <b>OS updates</b>.
      </span>
    );
  }

  if (apiReason.includes("Secret variable")) {
    return generateSecretErrMsg(err);
  }

  return apiReason || DEFAULT_ERROR_MESSAGE;
};
