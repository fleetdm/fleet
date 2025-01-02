import React from "react";
import { isAxiosError } from "axios";

import { getErrorReason } from "interfaces/errors";
import { ISoftwarePackage } from "interfaces/software";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";

const DEFAULT_ERROR_MESSAGE = "Couldn't edit software. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown, software: ISoftwarePackage) => {
  const isTimeout =
    isAxiosError(err) &&
    (err.response?.status === 504 || err.response?.status === 408);
  const reason = getErrorReason(err);

  if (isTimeout) {
    return "Couldn't upload. Request timeout. Please make sure your server and load balancer timeout is long enough.";
  } else if (reason.includes("Fleet couldn't read the version from")) {
    return (
      <>
        Couldn&apos;t edit <b>{software.name}</b>. {reason}.
        <CustomLink
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
          text="Learn more"
        />
      </>
    );
  } else if (reason.includes("selected package is")) {
    return (
      <>
        Couldn&apos;t edit <b>{software.name}</b>. {reason}
      </>
    );
  } else if (reason.includes("Secret variable")) {
    return generateSecretErrMsg(err).replace("Couldn't add", "Couldn't edit");
  }

  return reason || DEFAULT_ERROR_MESSAGE;
};
