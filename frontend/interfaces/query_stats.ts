import PropTypes, { number } from "prop-types";

import ILegacySchedulableQueryStats, {
  ISchedulableQueryStats,
} from "./schedulable_query";

export default PropTypes.shape({
  scheduled_query_name: PropTypes.string,
  scheduled_query_id: PropTypes.number,
  query_name: PropTypes.string,
  description: PropTypes.string,
  pack_name: PropTypes.string,
  pack_id: PropTypes.number,
  average_memory: number,
  denylisted: PropTypes.bool,
  executions: PropTypes.number,
  interval: PropTypes.number,
  last_executed: PropTypes.string,
  output_size: PropTypes.number,
  system_time: PropTypes.number,
  user_time: PropTypes.number,
  wall_time: PropTypes.number,
  stats: ILegacySchedulableQueryStats,
});

export interface IQueryStats {
  scheduled_query_name: string;
  scheduled_query_id: number;
  query_name: string;
  discard_data: boolean;
  last_fetched: string | null; // timestamp
  automations_enabled: boolean;
  description: string;
  pack_name: string;
  pack_id: number;
  average_memory: number;
  denylisted: boolean;
  executions: number;
  interval: number;
  last_executed: string;
  output_size?: number;
  system_time: number;
  user_time: number;
  wall_time?: number;
  stats?: ISchedulableQueryStats;
}
