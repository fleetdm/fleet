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
  const myDeviceInstructionsText = /Disk encryption: Requires action from the end user\. Ask the user to follow/;

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

  it("tells the admin to send the end user to My device when a BitLocker PIN is missing", () => {
    render(
      <HostDetailsBanners
        hostPlatform="windows"
        mdmEnrollmentStatus="On (manual)"
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionOSSetting={{
          status: "action_required",
          detail: "",
          action_required: "create_pin",
        }}
      />
    );

    expect(screen.getByText(myDeviceInstructionsText)).toBeInTheDocument();
  });

  it("tells the admin to ask for a restart when the repair is waiting on one", () => {
    render(
      <HostDetailsBanners
        hostPlatform="windows"
        mdmEnrollmentStatus="On (manual)"
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionOSSetting={{
          status: "action_required",
          detail: "",
          action_required: "restart",
        }}
      />
    );

    expect(screen.getByText(/Requires a restart/)).toBeInTheDocument();
    // The end user has nothing to do on My device here, so that instruction must not appear.
    expect(
      screen.queryByText(myDeviceInstructionsText)
    ).not.toBeInTheDocument();
  });

  it("renders no banner for a Windows host the end user cannot help", () => {
    render(
      <HostDetailsBanners
        hostPlatform="windows"
        mdmEnrollmentStatus="On (manual)"
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionOSSetting={{
          status: "action_required",
          detail:
            "BitLocker protection is off. Fleet could not turn it back on: the TPM is not ready",
        }}
      />
    );

    expect(
      screen.queryByText(myDeviceInstructionsText)
    ).not.toBeInTheDocument();
  });

  it("hides the Turn on MDM banner for never-updated devices", () => {
    const turnOnMdmText = /To enforce settings, OS updates, disk encryption, and more/;

    render(
      <HostDetailsBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        connectedToFleetMdm={false}
        macDiskEncryptionStatus={null}
        detailUpdatedAt="0001-01-01T00:00:00Z"
      />
    );

    expect(screen.queryByText(turnOnMdmText)).not.toBeInTheDocument();
  });

  it("renders the Turn on MDM banner for unenrolled macOS hosts that have updated its detail", () => {
    const turnOnMdmText = /To enforce settings, OS updates, disk encryption, and more/;

    render(
      <HostDetailsBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        connectedToFleetMdm={false}
        macDiskEncryptionStatus={null}
        detailUpdatedAt="2025-01-15T10:00:00Z"
      />
    );

    expect(screen.getByText(turnOnMdmText)).toBeInTheDocument();
  });
});
