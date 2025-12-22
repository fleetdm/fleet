import React from "react";
import { isAxiosError } from "axios";

import { getErrorReason } from "interfaces/errors";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";

import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";

import {
  ADD_SOFTWARE_ERROR_PREFIX,
  DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE,
  REQUEST_TIMEOUT_ERROR_MESSAGE,
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
} from "../helpers";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (err: unknown) => {
  const isTimeout =
    isAxiosError(err) &&
    (err.response?.status === 504 || err.response?.status === 408);
  const reason = getErrorReason(err);

  if (isTimeout) {
    return REQUEST_TIMEOUT_ERROR_MESSAGE;
  }

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    const alreadyAvailableMessage = formatAlreadyAvailableInstallMessage(
      reason
    );
    if (alreadyAvailableMessage) {
      return alreadyAvailableMessage;
    }
  }

  if (reason.includes("Secret variable")) {
    return generateSecretErrMsg(err);
  }

  if (reason.includes("Unable to extract necessary metadata")) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} Unable to extract necessary metadata.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/package-metadata-extraction`}
          text="Learn more"
          newTab
          variant="flash-message-link"
        />
      </>
    );
  }
  if (reason.includes("not a valid .tar.gz archive")) {
    return (
      <>
        This is not a valid .tar.gz archive.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/tarball-archives`}
          text="Learn more"
          newTab
          variant="flash-message-link"
        />
      </>
    );
  }

  if (reason) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(reason)}`;
  }

  return DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE;
};
