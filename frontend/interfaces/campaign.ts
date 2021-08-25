import PropTypes from "prop-types";
import { IHost } from "./host";

export default PropTypes.shape({
  hosts_count: PropTypes.shape({
    total: PropTypes.number,
    successful: PropTypes.number,
    failed: PropTypes.number,
  }),
  id: PropTypes.number,
  online: PropTypes.number,
});

export interface ICampaignQueryResult {
  build_distro: string;
  build_platform: string;
  config_hash: string;
  config_valid: string;
  extensions: string;
  host_hostname: string;
  instance_id: string;
  pid: string;
  platform_mask: string;
  start_time: string;
  uuid: string;
  version: string;
  watcher: string;
}

export interface ICampaign {
  Metrics?: {
    [key: string]: any;
  };
  created_at: string;
  errors: any;
  hosts: IHost[];
  hosts_count: {
    total: number;
    successful: number;
    failed: number;
  };
  id: number;
  query_id: number;
  query_results: ICampaignQueryResult[];
  status: string;
  totals: {
    count: number;
    missing_in_action: number;
    offline: number;
    online: number;
  };
  updated_at: string;
  user_id: number;
}

export interface ICampaignState {
  campaign: ICampaign;
  observerShowSql: boolean;
  queryIsRunning: boolean;
  queryPosition: {
    [key: string]: any;
  };
  queryResultsToggle: any;
  runQueryMilliseconds: number;
  selectRelatedHostTarget: boolean;
  targetsCount: number;
  targetsError: any;
}
