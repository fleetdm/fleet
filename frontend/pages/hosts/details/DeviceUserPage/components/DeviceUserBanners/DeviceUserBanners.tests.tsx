import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import DeviceUserBanners from "./DeviceUserBanners";

describe("Device User Banners", () => {
  const unableToEnrollIntoMdmExpectedText = /This device isn't eligible for MDM because it isn't assigned to your organization by Apple Business./;
  const turnOnMdmExpcetedText = /Mobile device management \(MDM\) is off\./;
  const resetNonLinuxDiskEncryptKeyExpectedText = /Disk encryption: Log out of your device or restart it to safeguard your data in case your device is lost or stolen\./;
  const diskEncryptionOffExpectedText = /Disk encryption: Disk encryption is turned off\. Contact your IT admin for additional instructions\./;
  const createNewLinuxDiskEncryptKeyExpectedText = /Disk encryption: Create a new disk encryption key\. This lets your organization help you unlock your device if you forget your passphrase\./;
  const createPINExepectedText = /Disk encryption: Create a BitLocker PIN to protect your data/;
  // <strong>Refetch</strong> splits the sentence, so match the run of text that follows it.
  const refetchToClearExpectedText = /to clear this banner/;

  it("renders the turn on mdm banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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

  it("renders the log out banner when FileVault is deferred to the next login", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
        diskEncryptionActionRequired="log_out"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(
      screen.getByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(diskEncryptionOffExpectedText)
    ).not.toBeInTheDocument();
  });

  it("tells the end user to contact IT when nothing enforces disk encryption", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (company-owned)"
        mdmEnabledAndConfigured
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
        diskEncryptionActionRequired="turn_on_encryption"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(diskEncryptionOffExpectedText)).toBeInTheDocument();
    expect(
      screen.queryByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).not.toBeInTheDocument();
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(createPINExepectedText)).toBeInTheDocument();
    // This fleetd cannot be handed the PIN, so the end user sets it themselves and has to refetch afterwards.
    expect(screen.getByText(refetchToClearExpectedText)).toBeInTheDocument();
  });

  it("drops the refetch instruction when fleetd can set the PIN", () => {
    render(
      <DeviceUserBanners
        hostPlatform="windows"
        diskEncryptionOSSetting={{
          status: "action_required",
          detail: "",
          action_required: "create_pin",
          fleetd_can_set_pin: true,
        }}
        diskIsEncrypted
        diskEncryptionKeyAvailable={false}
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(createPINExepectedText)).toBeInTheDocument();
    // Fleet clears the banner once its agent reports the PIN, so asking for a refetch would be busywork.
    expect(screen.queryByText(refetchToClearExpectedText)).toBeNull();
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        mdmEnabledAndConfigured
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment={false}
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

  it("renders the unable to enroll into MDM banner when only AB enrollment is allowed and not DEP assigned", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        depAssignedToFleet={false}
        onlyAllowAppleBusinessEnrollment
        connectedToFleetMdm={false}
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        detailUpdatedAt="2025-01-15T10:00:00Z"
        onTriggerEscrowLinuxKey={noop}
        onClickCreatePIN={noop}
        onClickTurnOnMdm={noop}
      />
    );

    expect(
      screen.getByText(unableToEnrollIntoMdmExpectedText)
    ).toBeInTheDocument();
  });
});
