export interface IPolicy {
  id: number;
  query_id: number;
  query_name: string;
  team_id: number;
  passing_host_count: number;
  failing_host_count: number;
  last_run_time?: string;
}
