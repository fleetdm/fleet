import { ISetupSoftwareStatus } from "interfaces/software";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown, hostName: string) => {
  return `Host "${hostName}" ${DEFAULT_ERROR_MESSAGE}`;
};

export const getIsSettingUpSoftware = (
  statuses: ISetupSoftwareStatus[] | null | undefined
) => {
  // important to distinguish between undefined (hasn't received initial response yet) and null or
  if (statuses === undefined) {
    // wait for API response
    return true;
  }
  // empty arr
  if (statuses === null || statuses.length === 0) {
    // not configured or no software selected
    return false;
  }

  // TODO - confirm condition of completion
  return statuses.some((s) => s.status !== "success");
};
