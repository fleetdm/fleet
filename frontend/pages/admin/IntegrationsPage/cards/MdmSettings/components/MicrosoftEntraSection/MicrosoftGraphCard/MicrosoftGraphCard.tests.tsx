import React from "react";
import { screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import MicrosoftGraphCard from "./MicrosoftGraphCard";

const COPY = {
  connect:
    "Add a Microsoft Entra app registration to sync Windows Autopilot devices to Fleet as pending hosts.",
  connected: "Microsoft Graph connected.",
  invalid:
    "Microsoft Graph credential is invalid. Windows Autopilot devices won't sync to Fleet as pending hosts.",
  unavailable: "Couldn't load the Microsoft Graph connection status.",
};

const renderCard = (
  props: Partial<React.ComponentProps<typeof MicrosoftGraphCard>> = {}
) => {
  const onViewDetails = jest.fn();
  return {
    onViewDetails,
    ...renderWithSetup(
      <MicrosoftGraphCard
        credentialAdded={false}
        credentialInvalid={false}
        onViewDetails={onViewDetails}
        {...props}
      />
    ),
  };
};

describe("MicrosoftGraphCard", () => {
  it("prompts to connect when no credential is stored", async () => {
    const { user, onViewDetails } = renderCard();

    expect(screen.getByText(COPY.connect)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Connect" }));
    expect(onViewDetails).toHaveBeenCalled();
  });

  it("renders the connected state when a healthy credential is stored", async () => {
    const { user, onViewDetails } = renderCard({ credentialAdded: true });

    expect(screen.getByText(COPY.connected)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Edit" }));
    expect(onViewDetails).toHaveBeenCalled();
  });

  it("calls out an invalid credential rather than showing it as connected", () => {
    renderCard({ credentialAdded: true, credentialInvalid: true });

    expect(screen.getByText(COPY.invalid)).toBeInTheDocument();
    expect(screen.queryByText(COPY.connected)).not.toBeInTheDocument();
    // Still editable, so the admin can fix it from here.
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });

  it("reports unknown status rather than Connect when the lookup failed", () => {
    renderCard({ credentialStatusUnavailable: true });

    expect(screen.getByText(COPY.unavailable)).toBeInTheDocument();
    // Claiming "Connect" here would misreport a configured tenant as disconnected.
    expect(
      screen.queryByRole("button", { name: "Connect" })
    ).not.toBeInTheDocument();
  });
});
