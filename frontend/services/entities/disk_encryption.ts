import sendRequest from "services";

import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

// TODO - move disk encryption types like this to dedicated file
import { DiskEncryptionStatus } from "interfaces/mdm";

export interface IDiskEncryptionStatusAggregate {
  macos: number;
  windows: number;
  linux: number;
}

export type IDiskEncryptionSummaryResponse = Record<
  DiskEncryptionStatus,
  IDiskEncryptionStatusAggregate
>;

const diskEncryptionService = {
  getDiskEncryptionSummary: (teamId?: number) => {
    let { MDM_DISK_ENCRYPTION_SUMMARY: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }
    return sendRequest("GET", path);
  },
  updateDiskEncryption: (enableDiskEncryption: boolean, teamId?: number) => {
    const { UPDATE_DISK_ENCRYPTION } = endpoints;
    return sendRequest("POST", UPDATE_DISK_ENCRYPTION, {
      enable_disk_encryption: enableDiskEncryption,
      team_id: teamId,
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
