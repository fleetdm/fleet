import sendRequest from "services";

import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

// TODO - move disk encryption types like this to dedicated file
import { DiskEncryptionStatus } from "interfaces/mdm";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

export interface IDiskEncryptionStatusAggregate {
  macos: number;
  windows: number;
  linux: number;
}

export type IDiskEncryptionSummaryResponse = Record<
  DiskEncryptionStatus,
  IDiskEncryptionStatusAggregate
>;

/** Absent keys are left unchanged by the server, so callers send only the
 * platform being saved. */
export interface IUpdateDiskEncryptionFormData {
  macos_settings?: {
    enable_disk_encryption: boolean;
    enable_escrow_disk_encryption_key: boolean;
  };
  windows_settings?: {
    enable_disk_encryption: boolean;
    require_bitlocker_pin: boolean;
  };
  linux_settings?: {
    enable_escrow_disk_encryption_key: boolean;
  };
}

const diskEncryptionService = {
  getDiskEncryptionSummary: (teamId?: number) => {
    let { DISK_ENCRYPTION: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ fleet_id: teamId })}`;
    }
    return sendRequest("GET", path);
  },
  updateDiskEncryption: (
    formData: IUpdateDiskEncryptionFormData,
    teamId?: number
  ) => {
    const { UPDATE_DISK_ENCRYPTION } = endpoints;
    return sendRequest("POST", UPDATE_DISK_ENCRYPTION, {
      ...formData,
      // the server expects fleet_id to be omitted for "No fleet", not 0
      fleet_id: teamId === APP_CONTEXT_NO_TEAM_ID ? undefined : teamId,
    });
  },
  triggerLinuxDiskEncryptionKeyEscrow: (token: string) => {
    const { DEVICE_TRIGGER_LINUX_DISK_ENCRYPTION_KEY_ESCROW } = endpoints;
    return sendRequest(
      "POST",
      DEVICE_TRIGGER_LINUX_DISK_ENCRYPTION_KEY_ESCROW(token)
    );
  },
};

export default diskEncryptionService;
