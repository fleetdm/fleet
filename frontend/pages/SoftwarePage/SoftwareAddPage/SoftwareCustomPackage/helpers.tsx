import React from "react";
import { isAxiosError } from "axios";

import { getErrorReason } from "interfaces/errors";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";

import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";

const DEFAULT_ERROR_MESSAGE = "Couldn't add. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const isTimeout =
    isAxiosError(err) &&
    (err.response?.status === 504 || err.response?.status === 408);
  const reason = getErrorReason(err);

  if (isTimeout) {
    return "Couldn't upload. Request timeout. Please make sure your server and load balancer timeout is long enough.";
  } else if (reason.includes("Fleet couldn't read the version from")) {
    return (
      <>
        {reason}{" "}
        <CustomLink
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
          text="Learn more"
          iconColor="core-fleet-white"
        />
      </>
    );
  } else if (reason.includes("Secret variable")) {
    return generateSecretErrMsg(err);
  } else if (reason.includes("Unable to extract necessary metadata")) {
    return (
      <>
        Couldn&apos;t add. Unable to extract necessary metadata.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/package-metadata-extraction`}
          text="Learn more"
          newTab
        />
      </>
    );
  } else if (reason.includes("Fleet couldn't read the version from")) {
    return (
      <>
        {reason}{" "}
        <CustomLink
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
          text="Learn more"
          iconColor="core-fleet-white"
        />
      </>
    );
  }

  return reason || DEFAULT_ERROR_MESSAGE;
};
