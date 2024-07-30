import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import DeviceUserBanners from "./DeviceUserBanners";

describe("Device User Banners", () => {
  const turnOnMdmExpcetedText = /Mobile device management \(MDM\) is off\./;
  const logoutDiskEncryptExpectedText = /Disk encryption: Log out of your device or restart to turn on disk encryption\./;
  const resetKeyDiskEncryptExpcetedText = /Disk encryption: Reset your disk encryption key\./;

  it("renders the turn on mdm banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="Off"
        mdmEnabledAndConfigured
        mdmConnectedToFleet
        diskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTurnOnMdm={noop}
      />
    );
    expect(screen.getByText(turnOnMdmExpcetedText)).toBeInTheDocument();
  });

  it("renders the reset key for disk encryption banner correctly", () => {
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        mdmConnectedToFleet
        diskEncryptionStatus="action_required"
        diskEncryptionActionRequired="rotate_key"
        onTurnOnMdm={noop}
      />
    );
    expect(
      screen.getByText(resetKeyDiskEncryptExpcetedText)
    ).toBeInTheDocument();
  });

  it("renders only one banner in a priority order", () => {
    // set up to render rotate key disk encryption banner, which is 2nd in priority
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus="On (automatic)"
        mdmEnabledAndConfigured
        mdmConnectedToFleet
        diskEncryptionStatus="action_required"
        diskEncryptionActionRequired="rotate_key"
        onTurnOnMdm={noop}
      />
    );

    expect(screen.queryByText(turnOnMdmExpcetedText)).not.toBeInTheDocument();
    expect(screen.getByText(logoutDiskEncryptExpectedText)).toBeInTheDocument();
    expect(
      screen.queryByText(resetKeyDiskEncryptExpcetedText)
    ).not.toBeInTheDocument();
  });

  it("renders no banner correctly", () => {
    // setup so mdm is not enabled and configured.
    render(
      <DeviceUserBanners
        hostPlatform="darwin"
        mdmEnrollmentStatus={null}
        mdmEnabledAndConfigured={false}
        mdmConnectedToFleet={false}
        diskEncryptionStatus={null}
        diskEncryptionActionRequired={null}
        onTurnOnMdm={noop}
      />
    );

    expect(screen.queryByText(turnOnMdmExpcetedText)).not.toBeInTheDocument();
    expect(
      screen.queryByText(logoutDiskEncryptExpectedText)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(resetKeyDiskEncryptExpcetedText)
    ).not.toBeInTheDocument();
  });
});
