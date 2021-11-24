export interface IPolicy {
  id: number;
  name: string;
  query: string;
  description: string;
  author_id: number;
  author_name: string;
  author_email: string;
  resolution: string;
  passing_host_count: number;
  failing_host_count: number;
  team_id?: number;
}

export interface IPolicyFormData {
  description?: string | number | boolean | any[] | undefined;
  name?: string | number | boolean | any[] | undefined;
  query?: string | number | boolean | any[] | undefined;
}
