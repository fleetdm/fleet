export interface IPolicy {
  id: number;
  name: string;
  query: string;
  description: string;
  author_id: number;
  author_name: string;
  author_email: string;
  resolution: string;
  team_id?: number;
  created_at: string;
  updated_at: string;
}

// Used on the manage hosts page and other places where aggregate stats are displayed
export interface IPolicyStats extends IPolicy {
  passing_host_count: number;
  failing_host_count: number;
}

// Used on the host details page and other places where the status of individual hosts are displayed
export interface IHostPolicy extends IPolicy {
  response?: "pass" | "fail";
}

export interface IPolicyFormData {
  description?: string | number | boolean | any[] | undefined;
  name?: string | number | boolean | any[] | undefined;
  query?: string | number | boolean | any[] | undefined;
}
