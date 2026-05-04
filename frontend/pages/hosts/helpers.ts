import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

export const getHostStatusTooltipText = (status: string): string => {
  if (status === "online") {
    return "Online hosts will respond to a live report.";
  }
  if (status === DEFAULT_EMPTY_CELL_VALUE) {
    return "Device is pending enrollment in Apple Business and status is not yet available.";
  }
  return "Offline hosts won't respond to a live report because they may be shut down, asleep, or not connected to the internet.";
};

export const getHostStatus = (
  status: string,
  mdmEnrollmentStatus?: string
): string => {
  if (mdmEnrollmentStatus === "Pending") {
    return DEFAULT_EMPTY_CELL_VALUE;
  }

  return status || DEFAULT_EMPTY_CELL_VALUE;
};
