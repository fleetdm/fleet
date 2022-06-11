export interface IDataTableMDMFormat {
  status: string;
  hosts: number;
}

export interface IMunkiAggregate {
  version: string;
  hosts_count: number;
}

export interface IMDMAggregateStatus {
  enrolled_manual_hosts_count: number;
  enrolled_automated_hosts_count: number;
  unenrolled_hosts_count: number;
}

export interface IMacadminAggregate {
  macadmins: {
    counts_updated_at: string;
    munki_versions: IMunkiAggregate[];
    mobile_device_management_enrollment_status: IMDMAggregateStatus;
  };
}
