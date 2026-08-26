import React from "react";

import { getErrorReason } from "interfaces/errors";
import { IMdmConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
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

export type IMdmDiskEncryptionSource =
  | IMdmConfig
  | NonNullable<ITeamConfig["mdm"]>;

export const getDiskEncryptionSettings = (
  mdm?: IMdmDiskEncryptionSource
): IDiskEncryptionSettings => {
  // fall back to the deprecated top-level keys so configs from servers that
  // don't return the per-platform fields yet still render their effective
  // state
  const legacyEnabled = mdm?.enable_disk_encryption ?? false;
  const windowsEnabled =
    mdm?.windows_settings?.enable_disk_encryption ?? legacyEnabled;
  return {
    macOSEnabled: mdm?.apple_settings?.enable_disk_encryption ?? legacyEnabled,
    macOSEscrowEnabled:
      mdm?.apple_settings?.enable_escrow_disk_encryption_key ?? legacyEnabled,
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
