export interface IPolicy {
  id: number;
  query_id: number;
  query_name: string;
  query_description: string;
  query: string;
  team_id: number;
  resolution: string;
  passing_host_count: number;
  failing_host_count: number;
  last_run_time?: string;
}

export interface IPolicyFormData {
  query_description?: string | number | boolean | any[] | undefined;
  query_name?: string | number | boolean | any[] | undefined;
  query?: string | number | boolean | any[] | undefined;
}
