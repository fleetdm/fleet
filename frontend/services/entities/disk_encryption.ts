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

const diskEncryptionService = {
  getDiskEncryptionSummary: (teamId?: number) => {
    let { MDM_DISK_ENCRYPTION_SUMMARY: path } = endpoints;

    if (teamId) {
      path = `${path}?${buildQueryStringFromParams({ team_id: teamId })}`;
    }
    return sendRequest("GET", path);
  },
  updateDiskEncryption: (enableDiskEncryption: boolean, teamId?: number) => {
    // TODO - use same endpoint for both once issue with new endpoint for no team is resolved
    const {
      UPDATE_DISK_ENCRYPTION: teamsEndpoint,
      CONFIG: noTeamsEndpoint,
    } = endpoints;
    if (teamId === 0) {
      return sendRequest("PATCH", noTeamsEndpoint, {
        mdm: {
          enable_disk_encryption: enableDiskEncryption,
        },
      });
    }
    return sendRequest("POST", teamsEndpoint, {
      enable_disk_encryption: enableDiskEncryption,
      // TODO - it would be good to be able to use an API_CONTEXT_NO_TEAM_ID here, but that is
      // currently set to 0, which should actually be undefined since the server expects teamId ==
      // nil for no teams, not 0.
      team_id: teamId === APP_CONTEXT_NO_TEAM_ID ? undefined : teamId,
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
