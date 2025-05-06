import React from "react";
import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

export const UPLOAD_ERROR_MESSAGES = {
  wrongType: {
    condition: (reason: string) => reason.includes("invalid file type"),
    message: "Couldn’t upload. The file should be a package (.pkg).",
  },
  unsigned: {
    condition: (reason: string) => reason.includes("file is not"),
    message:
      "Couldn’t upload. The package must be signed. Click “Learn more” below to learn how to sign.",
  },
  noDistribution: {
    condition: (reason: string) =>
      reason.includes("Bootstrap package must be a distribution package"),
    message: (
      <>
        Couldn&apos;t upload. Bootstrap package must be a distribution package.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/macos-distribution-packages`}
          text="Learn more"
          newTab
          variant="flash-message-link"
        />
      </>
    ),
  },
  default: {
    condition: () => false,
    message: "Couldn’t upload. Please try again.",
  },
};

export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err.data.errors[0].reason;

  const error = Object.values(UPLOAD_ERROR_MESSAGES).find((errType) =>
    errType.condition(apiReason)
  );

  if (!error) {
    return UPLOAD_ERROR_MESSAGES.default.message;
  }

  return error.message;
};
