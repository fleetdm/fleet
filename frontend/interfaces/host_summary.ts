import PropTypes from "prop-types";

export default PropTypes.shape({
  online_count: PropTypes.number,
  offline_count: PropTypes.number,
  mia_count: PropTypes.number,
  new_count: PropTypes.number,
});

export interface IHostSummaryPlatforms {
  platform: string;
  hosts_count: number;
}

export interface IHostSummaryLabel {
  id: number;
  name: string;
  description: string;
  label_type: "regular" | "builtin";
}

export interface IHostSummary {
  all_linux_count: number;
  totals_hosts_count: number;
  platforms: IHostSummaryPlatforms[] | null;
  online_count: number;
  offline_count: number;
  mia_count: number;
  new_count: number;
  builtin_labels: IHostSummaryLabel[];
}
