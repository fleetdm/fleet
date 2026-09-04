import React from "react";

import { getErrorReason } from "interfaces/errors";
import CustomLink from "components/CustomLink";

const PRIVATE_KEY_LEARN_MORE_LINK =
  "https://fleetdm.com/learn-more-about/fleet-server-private-key";

export const getErrorMessage = (err: unknown) => {
  const reason = getErrorReason(err);

  if (reason.includes("Missing required private key")) {
    return (
      <>
        Couldn&apos;t enable disk encryption. Please configure a private key.{" "}
        <CustomLink
          url={PRIVATE_KEY_LEARN_MORE_LINK}
          text="Learn how"
          newTab
          variant="flash-message-link"
        />
      </>
    );
  }

  return (
    reason || "Could not update the disk encryption settings. Please try again."
  );
};

export interface IDiskEncryptionSettings {
  macOSEnabled: boolean;
  macOSEscrowEnabled: boolean;
  windowsEnabled: boolean;
  windowsPINRequired: boolean;
  linuxEscrowEnabled: boolean;
}

interface IAppleDiskEncryptionSettings {
  enable_disk_encryption?: boolean;
  enable_escrow_disk_encryption_key?: boolean;
}

/** The disk encryption fields shared by the global config, single-fleet, and
 * fleet-list mdm responses. Fleet list responses (`teams`) still spell the
 * Apple settings `macos_settings`; the others use `apple_settings`. */
export interface IMdmDiskEncryptionSource {
  /** @deprecated Virtual key: true only when all four settings are on. */
  enable_disk_encryption?: boolean;
  /** @deprecated Use `windows_settings.require_bitlocker_pin`. */
  windows_require_bitlocker_pin?: boolean;
  apple_settings?: IAppleDiskEncryptionSettings;
  macos_settings?: IAppleDiskEncryptionSettings;
  windows_settings?: {
    enable_disk_encryption?: boolean;
    require_bitlocker_pin?: boolean;
  };
  linux_settings?: {
    enable_escrow_disk_encryption_key?: boolean;
  };
}

export const getDiskEncryptionSettings = (
  mdm?: IMdmDiskEncryptionSource
): IDiskEncryptionSettings => {
  // fall back to the deprecated top-level keys so configs from servers that
  // don't return the per-platform fields yet still render their effective
  // state
  const legacyEnabled = mdm?.enable_disk_encryption ?? false;
  const appleSettings = mdm?.apple_settings ?? mdm?.macos_settings;
  const windowsEnabled =
    mdm?.windows_settings?.enable_disk_encryption ?? legacyEnabled;
  return {
    macOSEnabled: appleSettings?.enable_disk_encryption ?? legacyEnabled,
    macOSEscrowEnabled:
      appleSettings?.enable_escrow_disk_encryption_key ?? legacyEnabled,
    windowsEnabled,
    // the server rejects a PIN requirement without encryption, so a stale PIN
    // flag must not make it into the form
    windowsPINRequired:
      windowsEnabled &&
      (mdm?.windows_settings?.require_bitlocker_pin ??
        mdm?.windows_require_bitlocker_pin ??
        false),
    linuxEscrowEnabled:
      mdm?.linux_settings?.enable_escrow_disk_encryption_key ?? legacyEnabled,
  };
};

/** macOS enforce on, escrow off — hosts never send Fleet a key. */
export const isMacOSDiskEncryptionEnforceOnly = ({
  macOSEnabled,
  macOSEscrowEnabled,
}: IDiskEncryptionSettings) => macOSEnabled && !macOSEscrowEnabled;
