import { render, screen } from "@testing-library/react";
import { AxiosResponse } from "axios";
import React from "react";

import { IApiError } from "interfaces/errors";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";

describe("getErrorMessage", () => {
  it("falls back to the default message for a failure with no API body", () => {
    ["Network Error", undefined].forEach((err) =>
      expect(
        getErrorMessage((err as unknown) as AxiosResponse<IApiError>)
      ).toBe(UPLOAD_ERROR_MESSAGES.default.message)
    );
  });

  it("shows the upload URL request's reason and hides other unmapped reasons", () => {
    const err = (url: string) =>
      (({
        config: { url },
        data: {
          errors: [{ name: "size", reason: "The maximum file size is 1KiB." }],
        },
      } as unknown) as AxiosResponse<IApiError>);
    expect(getErrorMessage(err("/api/latest/fleet/staged_upload"))).toBe(
      "Couldn’t upload. The maximum file size is 1KiB."
    );
    expect(getErrorMessage(err("/api/latest/fleet/bootstrap"))).toBe(
      UPLOAD_ERROR_MESSAGES.default.message
    );
  });

  it("maps a Fleet API reason", () => {
    const err = {
      data: { errors: [{ name: "base", reason: "file is not signed" }] },
    } as AxiosResponse<IApiError>;
    expect(getErrorMessage(err)).toBe(UPLOAD_ERROR_MESSAGES.unsigned.message);
  });

  it("links the unsigned package error to the signing guide", () => {
    render(<>{UPLOAD_ERROR_MESSAGES.unsigned.message}</>);
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/sign-bootstrap-package"
    );
  });
});
