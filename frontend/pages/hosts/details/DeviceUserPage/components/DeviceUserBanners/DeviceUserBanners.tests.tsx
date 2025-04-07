import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import DeviceUserBanners from "./DeviceUserBanners";

describe("Device User Banners", () => {
  const turnOnMdmExpcetedText = /Mobile device management \(MDM\) is off\./;
  const resetNonLinuxDiskEncryptKeyExpectedText = /Disk encryption: Log out of your device or restart it to safeguard your data in case your device is lost or stolen\./;
  const createNewLinuxDiskEncryptKeyExpectedText = /Disk encryption: Create a new disk encryption key\. This lets your organization help you unlock your device if you forget your passphrase\./;

  it("renders the turn on mdm banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTurnOnMdm={noop}
        onTriggerEscrowLinuxKey={noop}
      />
    );
    expect(screen.getByText(turnOnMdmExpcetedText)).toBeInTheDocument();
  });

  it("renders the reset key for non-linux disk encryption banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        connectedToFleetMdm
        macDiskEncryptionStatus="action_required"
        diskEncryptionActionRequired="rotate_key"
        onTurnOnMdm={noop}
        onTriggerEscrowLinuxKey={noop}
      />
    );
    expect(
      screen.getByText(resetNonLinuxDiskEncryptKeyExpectedText)
    ).toBeInTheDocument();
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
        onTurnOnMdm={noop}
        onTriggerEscrowLinuxKey={noop}
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
        onTurnOnMdm={noop}
        onTriggerEscrowLinuxKey={noop}
      />
    );
    expect(
      screen.getByText(createNewLinuxDiskEncryptKeyExpectedText)
    ).toBeInTheDocument();
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
        onTurnOnMdm={noop}
        onTriggerEscrowLinuxKey={noop}
        diskEncryptionOSSetting={{ status: "verifying", detail: "" }}
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
        onTurnOnMdm={noop}
        onTriggerEscrowLinuxKey={noop}
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
});
