import { ISchedulableQueryStats } from "./schedulable_query";

export interface IQueryStats {
  scheduled_report_name: string;
  scheduled_report_id: number;
  report_name: string;
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
