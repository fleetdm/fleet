// legacy interfaces to maintain packs support
import PropTypes from "prop-types";
import ILegacySchedulableQueryStats, {
  ISchedulableQueryStats,
} from "interfaces/schedulable_query";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number.isRequired,
  pack_id: PropTypes.number.isRequired,
  name: PropTypes.string.isRequired,
  query_id: PropTypes.number.isRequired,
  query_name: PropTypes.string.isRequired,
  query: PropTypes.string.isRequired,
  interval: PropTypes.number.isRequired,
  snapshot: PropTypes.bool,
  removed: PropTypes.bool,
  platform: PropTypes.string,
  version: PropTypes.string,
  shard: PropTypes.number,
  denylist: PropTypes.bool,
  stats: ILegacySchedulableQueryStats,
});

export interface IPackQueryFormData {
  interval?: number;
  name?: string;
  shard?: number;
  query?: string;
  query_id?: number;
  pack_id?: number;
  logging_type?: string;
  removed?: boolean;
  snapshot?: boolean;
  platform?: string;
  version?: string;
}
export interface IScheduledQuery {
  created_at: string;
  updated_at: string;
  id: number;
  pack_id: number;
  name: string;
  query_id: number;
  query_name: string;
  query: string;
  interval: number;
  snapshot: boolean;
  removed: boolean;
  platform?: string;
  version?: string;
  shard?: number | undefined;
  denylist?: boolean;
  logging_type?: string;
  stats: ISchedulableQueryStats;
  team_id?: number;
}
export interface IEditScheduledQuery extends IScheduledQuery {
  type: "global_scheduled_query" | "team_scheduled_query";
}
export interface ILoadAllGlobalScheduledQueriesResponse {
  global_schedule: IScheduledQuery[];
}

export interface IStoredScheduledQueriesResponse {
  scheduled: IScheduledQuery[];
}

export interface IUpdateScheduledQuery {
  interval?: number;
  logging_type: string;
  platform?: string;
  version?: string;
  shard?: number;
}

export interface IUpdateTeamScheduledQuery extends IUpdateScheduledQuery {
  team_id: number;
}
