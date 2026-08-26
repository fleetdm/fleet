import React from "react";
import { screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import { getErrorMessage } from "./helpers";

const createApiError = (reason: string) => ({
  response: { data: { errors: [{ name: "base", reason }] } },
});

describe("getErrorMessage", () => {
  it("explains how to configure a private key when the server is missing one", () => {
    renderWithSetup(
      <>
        {getErrorMessage(
          createApiError("Missing required private key. Learn how to configure")
        )}
      </>
    );

    expect(
      screen.getByText(
        /Couldn't enable disk encryption\. Please configure a private key\./
      )
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Learn how/ })).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/fleet-server-private-key"
    );
  });

  it("returns the server's reason for other errors", () => {
    expect(getErrorMessage(createApiError("Windows MDM is not enabled"))).toBe(
      "Windows MDM is not enabled"
    );
  });

  it("falls back to a generic message when there is no reason", () => {
    expect(getErrorMessage(new Error("network"))).toBe(
      "Could not update the disk encryption settings. Please try again."
    );
  });
});
