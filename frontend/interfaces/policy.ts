export interface IPolicy {
  id: number;
  query_id: number;
  query_name: string;
  passing_host_count: number;
  failing_host_count: number;
}
