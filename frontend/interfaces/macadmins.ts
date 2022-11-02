export interface IDataTableMdmFormat {
  status: "Enrolled (manual)" | "Enrolled (automatic)" | "Unenrolled";
  hosts: number;
}

export interface IMunkiVersionsAggregate {
  version: string;
  hosts_count: number;
}

export interface IMunkiIssuesAggregate {
  id: number;
  name: string;
  type: "error" | "warning";
  hosts_count: number;
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

export interface IMacadminAggregate {
  macadmins: {
    counts_updated_at: string;
    munki_versions: IMunkiVersionsAggregate[];
    munki_issues: IMunkiIssuesAggregate[];
    mobile_device_management_enrollment_status: IMdmAggregateStatus;
    mobile_device_management_solution: IMdmSolution[] | null;
  };
}
