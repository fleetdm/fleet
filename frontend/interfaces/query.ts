import { ISchedulableQueryStats } from "./schedulable_query";

export interface IQuery {
  created_at: string;
  updated_at: string;
  id: number;
  name: string;
  description: string;
  query: string;
  saved: boolean;
  author_id: number;
  author_name: string;
  author_email: string;
  observer_can_run: boolean;
  stats?: ISchedulableQueryStats;
}
