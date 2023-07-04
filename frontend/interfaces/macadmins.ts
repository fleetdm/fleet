import { IMdmAggregateStatus, IMdmSolution } from "./mdm";

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

export interface IMacadminAggregate {
  macadmins: {
    counts_updated_at: string;
    munki_versions: IMunkiVersionsAggregate[];
    munki_issues: IMunkiIssuesAggregate[];
    mobile_device_management_enrollment_status: IMdmAggregateStatus;
    mobile_device_management_solution: IMdmSolution[] | null;
  };
}
