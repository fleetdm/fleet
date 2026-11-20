import { ILabelSummary } from "./label";

export interface IHostSummaryPlatforms {
  platform: string;
  hosts_count: number;
}

export interface IHostSummary {
  totals_hosts_count: number;
  platforms: IHostSummaryPlatforms[] | null;
  online_count: number;
  offline_count: number;
  new_count: number;
  missing_30_days_count?: number; // premium feature
  low_disk_space_count?: number; // premium feature
  dep_assign_error_count?: number; // premium feature
  builtin_labels: ILabelSummary[];
}
