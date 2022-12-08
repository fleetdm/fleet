export interface IMdmEnrollmentCardData {
  status: "Enrolled (manual)" | "Enrolled (automatic)" | "Unenrolled";
  hosts: number;
}

export interface IMdmAggregateStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
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
  unenrolled_hosts_count: number;
  hosts_count: number;
}

export interface IMdmSummaryResponse {
  counts_updated_at: string;
  mobile_device_management_enrollment_status: IMdmEnrollementStatus;
  mobile_device_management_solution: IMdmSolution[] | null;
}

export interface IHostMdmResponse {
  enrollment_status: string | null;
  server_url: string | null;
  name: string | null;
  id: number;
}
