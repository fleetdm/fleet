import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import HostDetailsBanners from "./HostDetailsBanners";

const render = createCustomRenderer({
  context: { app: { config: createMockConfig() } },
});

describe("Host Details Banners", () => {
  const logOutExpectedText = /Disk encryption: Requires action from the end user\. Ask the end user to log out of their device or restart it\./;
  const escrowedAutomaticallyExpectedText = /Disk encryption: FileVault key will be escrowed automatically on this host's next refetch\./;

  it("tells the admin the key is escrowed automatically for ADE-enrolled hosts", () => {
    render(
      <HostDetailsBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
      />
    );

    expect(
      screen.getByText(escrowedAutomaticallyExpectedText)
    ).toBeInTheDocument();
    expect(screen.queryByText(logOutExpectedText)).not.toBeInTheDocument();
  });

  // "On (company-owned)" is the current name for automatic enrollment; "On (automatic)"
  // is the legacy value the API still returns
  it("tells the admin the key is escrowed automatically for company-owned hosts", () => {
    render(
      <HostDetailsBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (company-owned)"
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
      />
    );

    expect(
      screen.getByText(escrowedAutomaticallyExpectedText)
    ).toBeInTheDocument();
  });

  it("tells the admin to ask the end user to log out for manually-enrolled hosts", () => {
    render(
      <HostDetailsBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (manual)"
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
      />
    );

    expect(screen.getByText(logOutExpectedText)).toBeInTheDocument();
    expect(
      screen.queryByText(escrowedAutomaticallyExpectedText)
    ).not.toBeInTheDocument();
  });

  it("renders no disk encryption banner when the key is not in an action required state", () => {
    render(
      <HostDetailsBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        connectedToFleetMdm
        macDiskEncryptionStatus="verifying"
      />
    );

    expect(
      screen.queryByText(escrowedAutomaticallyExpectedText)
    ).not.toBeInTheDocument();
    expect(screen.queryByText(logOutExpectedText)).not.toBeInTheDocument();
  });
});
