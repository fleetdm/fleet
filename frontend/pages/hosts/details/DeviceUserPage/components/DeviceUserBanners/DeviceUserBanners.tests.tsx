import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import DeviceUserBanners from "./DeviceUserBanners";

describe("Device User Banners", () => {
  const turnOnMdmExpcetedText = /Mobile device management \(MDM\) is off\./;
  const resetNonLinuxDiskEncryptKeyExpectedText = /Disk encryption: Log out of your device or restart it to safeguard your data in case your device is lost or stolen\./;
  const adeDiskEncryptKeyExpectedText = /Disk encryption: Refetch to ensure data is safeguarded in case your device is lost or stolen\. If this banner persists, contact your IT admin\./;
  const createNewLinuxDiskEncryptKeyExpectedText = /Disk encryption: Create a new disk encryption key\. This lets your organization help you unlock your device if you forget your passphrase\./;
  const createPINExepectedText = /Disk encryption: Create a BitLocker PIN to safeguard your data/;

  it("renders the turn on mdm banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        detailUpdatedAt="2025-01-15T10:00:00Z"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(turnOnMdmExpcetedText)).toBeInTheDocument();
  });

  it("renders the reset key for non-linux disk encryption banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (manual)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
        diskEncryptionActionRequired="rotate_key"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(
      screen.getByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).toBeInTheDocument();
  });

  it("renders the refetch disk encryption banner for ADE-enrolled hosts", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
        diskEncryptionActionRequired="rotate_key"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(adeDiskEncryptKeyExpectedText)).toBeInTheDocument();
    expect(
      screen.queryByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).not.toBeInTheDocument();
  });

  // "On (company-owned)" is the current name for automatic enrollment; "On (automatic)"
  // is the legacy value the API still returns
  it("renders the refetch disk encryption banner for company-owned hosts", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (company-owned)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
        diskEncryptionActionRequired="rotate_key"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(adeDiskEncryptKeyExpectedText)).toBeInTheDocument();
  });
  it("renders the create new linux disk encryption key banner correctly for Ubuntu", () => {
    render(
      <DeviceUserBanners
        hostPlatform="ubuntu"
        diskEncryptionOSSetting={{ status: "action_required", detail: "" }}
        diskIsEncrypted
        // explicit for testing purposes
        diskEncryptionKeyAvailable={false}
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(
      screen.getByText(createNewLinuxDiskEncryptKeyExpectedText)
    ).toBeInTheDocument();
  });
  it("renders the create new linux disk encryption key banner correctly for Fedora", () => {
    render(
      <DeviceUserBanners
        hostPlatform="rhel"
        hostOsVersion="somethingsomethingfedorasomething"
        diskEncryptionOSSetting={{ status: "action_required", detail: "" }}
        diskIsEncrypted
        // explicit for testing purposes
        diskEncryptionKeyAvailable={false}
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(
      screen.getByText(createNewLinuxDiskEncryptKeyExpectedText)
    ).toBeInTheDocument();
  });

  it("renders the create PIN banner correctly for Windows", () => {
    render(
      <DeviceUserBanners
        hostPlatform="windows"
        diskEncryptionOSSetting={{
          status: "action_required",
          detail: "",
          action_required: "create_pin",
        }}
        diskIsEncrypted
        // explicit for testing purposes
        diskEncryptionKeyAvailable={false}
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(createPINExepectedText)).toBeInTheDocument();
  });

  it("asks the end user to restart when the repair is waiting on one", () => {
    render(
      <DeviceUserBanners
        hostPlatform="windows"
        diskEncryptionOSSetting={{
          status: "action_required",
          detail: "",
          action_required: "restart",
        }}
        diskIsEncrypted
        diskEncryptionKeyAvailable={false}
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(/Restart your device/)).toBeInTheDocument();
    // A PIN is not what unblocks this, so offering one would send the end user down the wrong path.
    expect(screen.queryByText(createPINExepectedText)).toBeNull();
  });

  // BitLocker reaches action_required for problems the end user cannot touch: an unready TPM, or policy forbidding a
  // TPM-only protector. Offering "Create PIN" there points them at a fix that cannot work, and the promised Refetch
  // never clears the banner because the cause persists.
  it("renders no create PIN banner for Windows when the end user cannot act", () => {
    render(
      <DeviceUserBanners
        hostPlatform="windows"
        diskEncryptionOSSetting={{
          status: "action_required",
          detail:
            "BitLocker protection is off. Fleet could not turn it back on: the TPM is not ready",
        }}
        diskIsEncrypted
        diskEncryptionKeyAvailable={false}
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.queryByText(createPINExepectedText)).toBeNull();
  });

  it("renders no banner correctly for a mac that is verifying its disk encryption", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (manual)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus="verifying"
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        diskEncryptionOSSetting={{ status: "verifying", detail: "" }}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );

    expect(screen.queryByText(turnOnMdmExpcetedText)).not.toBeInTheDocument();
    expect(
      screen.queryByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).not.toBeInTheDocument();
  });
  it("renders no banner correctly for a mac without MDM set up", () => {
    // setup so mdm is not enabled and configured.
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus={null}
        mdmEnabledAndConfigured={false}
        connectedToFleetMdm={false}
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );

    expect(screen.queryByText(turnOnMdmExpcetedText)).not.toBeInTheDocument();
    expect(
      screen.queryByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).not.toBeInTheDocument();
  });

  it("hides the Turn on MDM banner for never-fetched devices", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        connectedToFleetMdm={false}
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        detailUpdatedAt="0001-01-01T00:00:00Z"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );

    expect(screen.queryByText(turnOnMdmExpcetedText)).not.toBeInTheDocument();
  });

  it("renders the Turn on MDM banner for unenrolled macOS hosts that have updated its detail", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        connectedToFleetMdm={false}
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        detailUpdatedAt="2025-01-15T10:00:00Z"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );

    expect(screen.getByText(turnOnMdmExpcetedText)).toBeInTheDocument();
  });
});
