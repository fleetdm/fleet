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

export const MDM_STATUS = {
  "On (manual)": "manual",
  "On (automatic)": "automatic",
  Off: "unenrolled",
  Pending: "pending",
};

export type MdmStatus = keyof typeof MDM_STATUS;

export interface IMdmStatusCardData {
  status: MdmStatus;
  hosts: number;
}

export interface IMdmAggregateStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
  pending_hosts_count?: number;
}

export interface IMdmSolution {
  id: number;
  name: string | null;
  server_url: string;
  hosts_count: number;
}

interface IMdmStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
  pending_hosts_count?: number;
  hosts_count: number;
}

export interface IMdmSummaryResponse {
  counts_updated_at: string;
  mobile_device_management_enrollment_status: IMdmStatus;
  mobile_device_management_solution: IMdmSolution[] | null;
}
