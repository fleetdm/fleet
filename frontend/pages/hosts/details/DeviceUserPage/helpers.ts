import { ISetupSoftwareStatus } from "interfaces/software";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown, hostName: string) => {
  return `Host "${hostName}" ${DEFAULT_ERROR_MESSAGE}`;
};

export const getIsSettingUpSoftware = (
  statuses: ISetupSoftwareStatus[] | null | undefined
) => {
  if (!statuses || statuses.length === 0) {
    // not configured or no software selected
    return false;
  }

  return statuses.some((s) => ["pending", "running"].includes(s.status));
};
