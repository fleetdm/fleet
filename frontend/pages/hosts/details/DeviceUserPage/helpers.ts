import { ISetupStep } from "interfaces/setup";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown, hostName: string) => {
  return `Host "${hostName}" ${DEFAULT_ERROR_MESSAGE}`;
};

export const hasRemainingSetupSteps = (
  statuses: ISetupStep[] | null | undefined
) => {
  if (!statuses || statuses.length === 0) {
    // not configured or no software selected
    return false;
  }

  return statuses.some((s) => ["pending", "running"].includes(s.status));
};

export const getFailedSoftwareInstall = (
  statuses: ISetupStep[] | null | undefined
): ISetupStep | null => {
  if (!statuses || statuses.length === 0) {
    // not configured or no software selected
    return null;
  }

  const failedSoftware =
    statuses.filter((s) => s.type === "software" && s.status === "failure") ||
    [];
  if (failedSoftware.length === 0) {
    return null;
  }
  // Find the first one with an error message, otherwise return the first one.
  const firstWithError = failedSoftware.find((s) => s.error);
  return firstWithError ?? failedSoftware[0];
};
