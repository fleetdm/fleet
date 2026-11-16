import { render, screen } from "@testing-library/react";
import React from "react";

import { getConditionalAccessErrorMessage } from "./helpers";

describe("getConditionalAccessErrorMessage", () => {
  it("links to the private key docs when the server private key is missing", () => {
    render(
      <>
        {getConditionalAccessErrorMessage(
          "download",
          "Fleet server private key is not configured. Learn more: https://fleetdm.com/learn-more-about/fleet-server-private-key"
        )}
      </>
    );

    expect(screen.getByText(/Couldn't download/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Learn how/ })).toHaveAttribute(
      "href",
      expect.stringContaining("fleet-server-private-key")
    );
  });

  it("includes the action and the server's reason", () => {
    const message = getConditionalAccessErrorMessage(
      "load Apple profile",
      "missing SCEP challenge"
    );

    expect(message).toEqual(expect.stringContaining("load Apple profile"));
    expect(message).toEqual(expect.stringContaining("missing SCEP challenge"));
  });

  it("falls back to a generic message when there's no reason", () => {
    expect(getConditionalAccessErrorMessage("download", "")).toBe(
      "Something's gone wrong. Please try again."
    );
  });
});
