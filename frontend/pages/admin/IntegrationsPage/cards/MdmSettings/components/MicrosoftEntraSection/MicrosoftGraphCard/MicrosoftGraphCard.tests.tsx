import React from "react";
import noop from "lodash/noop";

import { render, screen } from "@testing-library/react";

import MicrosoftGraphCard from "./MicrosoftGraphCard";

describe("MicrosoftGraphCard", () => {
  it("prompts to connect when no credential is stored", async () => {
    render(
      <MicrosoftGraphCard
        credentialAdded={false}
        credentialInvalid={false}
        viewDetails={noop}
      />
    );

    expect(
      await screen.findByRole("button", { name: "Connect" })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/sync Windows Autopilot devices to Fleet/i)
    ).toBeInTheDocument();
  });

  it("renders the connected state when a healthy credential is stored", async () => {
    render(
      <MicrosoftGraphCard
        credentialAdded
        credentialInvalid={false}
        viewDetails={noop}
      />
    );

    expect(
      await screen.findByRole("button", { name: "Edit" })
    ).toBeInTheDocument();
    expect(screen.getByText("Microsoft Graph connected.")).toBeInTheDocument();
  });

  it("calls out an invalid credential rather than showing it as connected", async () => {
    render(
      <MicrosoftGraphCard
        credentialAdded
        credentialInvalid
        viewDetails={noop}
      />
    );

    expect(
      await screen.findByText(/client secret is invalid/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Microsoft Graph connected.")
    ).not.toBeInTheDocument();
    // Still editable, so the admin can fix it from here.
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });
});
