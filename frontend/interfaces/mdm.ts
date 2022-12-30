export interface IMdmApple {
  common_name: string;
  serial_number: string;
  issuer: string;
  renew_date: string;
}

export interface IMdmAppleBm {
  default_team?: string;
  apple_id: string;
  organization_name: string;
  mdm_server_url: string;
  renew_date: string;
}

export interface IMdmAggregateStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  pending_automated_hosts_count: number;
  unenrolled_hosts_count: number;
}

export interface IMdmSolution {
  id: number;
  name: string | null;
  server_url: string;
  hosts_count: number;
}

interface IMdmEnrollementStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  pending_hosts_count: number;
  unenrolled_hosts_count: number;
  hosts_count: number;
}

export interface IMdmSummaryResponse {
  counts_updated_at: string;
  mobile_device_management_enrollment_status: IMdmEnrollementStatus;
  mobile_device_management_solution: IMdmSolution[] | null;
}

export const MDM_STATUS_DISPLAY_TEXT = [
  "On (manual)",
  "On (automatic)",
  "Pending",
  "Off",
] as const;

export type IMdmStatusDisplayText = typeof MDM_STATUS_DISPLAY_TEXT[number];

export interface IMdmEnrollmentCardData {
  status: IMdmStatusDisplayText;
  hosts: number;
}

export const MDM_STATUS_CONVERSION: Record<string, IMdmStatusDisplayText> = {
  pending: "Pending",
  unenrolled: "Off",
  automatic: "On (automatic)",
  "enrolled (automated)": "On (automatic)",
  manual: "On (manual)",
  "enrolled (manual)": "On (manual)",
} as const;

export const formatMdmStatusForDisplay = (
  status: string
): IMdmStatusDisplayText | undefined => {
  if (!status) {
    return undefined;
  }
  return MDM_STATUS_CONVERSION[status.toLowerCase()];
};

export const MDM_STATUS_QUERY_PARAMS = [
  "pending",
  "unenrolled",
  "automatic",
  "manual",
] as const;

export type IMdmStatusQueryParam = typeof MDM_STATUS_QUERY_PARAMS[number];

/**
 * formatMdmStatusForUrl attempts to convert display-formatted MDM status text to a valid
 * value for the `mdm_enrollment_status` query param. If the provided string cannot be converted,
 * formatMdmStatusForUrl returns `undefined`.
 */
export const formatMdmStatusForUrl = (status: IMdmStatusDisplayText) => {
  const match = Object.keys(MDM_STATUS_CONVERSION).find(
    (k) =>
      MDM_STATUS_CONVERSION[k].toLowerCase() === status.toLowerCase() &&
      MDM_STATUS_QUERY_PARAMS.find((p) => p.toLowerCase() === k.toLowerCase())
  );
  return match ? (match as IMdmStatusQueryParam) : undefined;
};
