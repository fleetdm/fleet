import { IFormField } from "./form_field";
import { IPack } from "./pack";
import { ISchedulableQuery, ISchedulableQueryStats } from "./schedulable_query";

export interface IEditQueryFormData {
  description?: string | number | boolean | undefined;
  name?: string | number | boolean | undefined;
  query?: string | number | boolean | undefined;
  observer_can_run?: string | number | boolean | undefined;
  automations_enabled?: boolean;
}

export interface IStoredQueryResponse {
  query: ISchedulableQuery;
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
  stats?: ISchedulableQueryStats;
}

export interface IEditQueryFormFields {
  description: IFormField;
  name: IFormField;
  query: IFormField;
  observer_can_run: IFormField;
}
