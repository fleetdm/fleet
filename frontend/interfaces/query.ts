import { IFormField } from "./form_field";
import { IPack } from "./pack";
import { IScheduledQueryStats } from "./scheduled_query_stats";

export interface IQueryFormData {
  description?: string | number | boolean | undefined;
  name?: string | number | boolean | undefined;
  query?: string | number | boolean | undefined;
  observer_can_run?: string | number | boolean | undefined;
}

export interface IStoredQueryResponse {
  query: IQuery;
}

export interface IFleetQueriesResponse {
  queries: IQuery[];
}

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
  packs: IPack[];
  stats?: IScheduledQueryStats;
}

export interface IQueryFormFields {
  description: IFormField;
  name: IFormField;
  query: IFormField;
  observer_can_run: IFormField;
}
