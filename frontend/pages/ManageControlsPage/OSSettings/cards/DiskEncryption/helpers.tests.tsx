import React from "react";
import { screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import {
  getDiskEncryptionSettings,
  getErrorMessage,
  isMacOSDiskEncryptionEnforceOnly,
} from "./helpers";

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

describe("getDiskEncryptionSettings", () => {
  const perPlatform = {
    enable_disk_encryption: false,
    windows_require_bitlocker_pin: false,
    windows_settings: {
      enable_disk_encryption: true,
      require_bitlocker_pin: true,
    },
    linux_settings: { enable_escrow_disk_encryption_key: true },
  };

  it("reads the Apple settings from a single-fleet or config response", () => {
    expect(
      getDiskEncryptionSettings({
        ...perPlatform,
        apple_settings: {
          enable_disk_encryption: true,
          enable_escrow_disk_encryption_key: false,
        },
      })
    ).toEqual({
      macOSEnabled: true,
      macOSEscrowEnabled: false,
      windowsEnabled: true,
      windowsPINRequired: true,
      linuxEscrowEnabled: true,
    });
  });

  it("reads the Apple settings from a fleet list response, which still spells them macos_settings", () => {
    const settings = getDiskEncryptionSettings({
      ...perPlatform,
      macos_settings: {
        enable_disk_encryption: true,
        enable_escrow_disk_encryption_key: false,
      },
    });

    expect(settings.macOSEnabled).toBe(true);
    expect(settings.macOSEscrowEnabled).toBe(false);
  });

  it("falls back to the deprecated top-level key when per-platform fields are absent", () => {
    const settings = getDiskEncryptionSettings({
      enable_disk_encryption: true,
      windows_require_bitlocker_pin: true,
    });

    expect(settings).toEqual({
      macOSEnabled: true,
      macOSEscrowEnabled: true,
      windowsEnabled: true,
      windowsPINRequired: true,
      linuxEscrowEnabled: true,
    });
  });

  it("ignores a PIN requirement when Windows disk encryption is off", () => {
    const settings = getDiskEncryptionSettings({
      ...perPlatform,
      windows_settings: {
        enable_disk_encryption: false,
        require_bitlocker_pin: true,
      },
    });

    expect(settings.windowsEnabled).toBe(false);
    expect(settings.windowsPINRequired).toBe(false);
  });
});

describe("isMacOSDiskEncryptionEnforceOnly", () => {
  const settings = {
    windowsEnabled: false,
    windowsPINRequired: false,
    linuxEscrowEnabled: false,
  };

  it.each([
    [true, false, true],
    [true, true, false],
    [false, true, false],
    [false, false, false],
  ])(
    "enforce %s / escrow %s -> %s",
    (macOSEnabled, macOSEscrowEnabled, expected) => {
      expect(
        isMacOSDiskEncryptionEnforceOnly({
          ...settings,
          macOSEnabled,
          macOSEscrowEnabled,
        })
      ).toBe(expected);
    }
  );
});
