// for legacy legacy query stats interface
import PropTypes from "prop-types";

import { IFormField } from "./form_field";
import { IPack } from "./pack";
import {
  CommaSeparatedPlatformString,
  QueryablePlatform,
  SelectedPlatform,
} from "./platform";

// Query itself
export interface ISchedulableQuery {
  created_at: string;
  updated_at: string;
  id: number;
  name: string;
  description: string;
  query: string;
  team_id: number | null;
  interval: number;
  platform: CommaSeparatedPlatformString; // Might more accurately be called `platforms_to_query` or `targeted_platforms` – comma-separated string of platforms to query, default all platforms if omitted
  min_osquery_version: string;
  automations_enabled: boolean;
  logging: QueryLoggingOption;
  saved: boolean;
  author_id: number;
  author_name: string;
  author_email: string;
  observer_can_run: boolean;
  discard_data: boolean;
  packs: IPack[];
  stats: ISchedulableQueryStats;
  editingExistingQuery?: boolean;
}

export interface IEnhancedQuery extends ISchedulableQuery {
  performance: string;
  targetedPlatforms: QueryablePlatform[];
}
export interface ISchedulableQueryStats {
  user_time_p50?: number | null;
  user_time_p95?: number | null;
  system_time_p50?: number | null;
  system_time_p95?: number | null;
  total_executions?: number;
}

// legacy
export default PropTypes.shape({
  user_time_p50: PropTypes.number,
  user_time_p95: PropTypes.number,
  system_time_p50: PropTypes.number,
  system_time_p95: PropTypes.number,
  total_executions: PropTypes.number,
});

// API shapes

// Get a query by id
/** GET /api/v1/fleet/queries/{id}` */
export interface IGetQueryResponse {
  query: ISchedulableQuery;
}

// List global or team queries
/**  GET /api/v1/fleet/queries?order_key={column_from_queries_table}&order_direction={asc|desc}&team_id={team_id} */
export interface IListQueriesResponse {
  queries: ISchedulableQuery[];
}

export interface IQueryKeyQueriesLoadAll {
  scope: "queries";
  teamId?: number;
  page?: number;
  perPage?: number;
  query?: string;
  orderDirection?: "asc" | "desc";
  orderKey?: string;
  mergeInherited?: boolean;
  targetedPlatform?: SelectedPlatform;
}
// Create a new query
/** POST /api/v1/fleet/queries */
export interface ICreateQueryRequestBody {
  name: string;
  query: string;
  description?: string;
  observer_can_run?: boolean;
  discard_data?: boolean;
  team_id?: number; // global query if ommitted
  interval?: number; // default 0 means never run
  platform?: CommaSeparatedPlatformString; // Might more accurately be called `platforms_to_query` – comma-separated string of platforms to query, default all platforms if omitted
  min_osquery_version?: string; // default all versions if ommitted
  automations_enabled?: boolean; // whether to send data to the configured log destination according to the query's `interval`. Default false if ommitted.
  logging?: QueryLoggingOption;
}

// response is ISchedulableQuery

// Modify a query by id
/** PATCH /api/v1/fleet/queries/{id} */
export interface IModifyQueryRequestBody
  extends Omit<ICreateQueryRequestBody, "name" | "query"> {
  id?: number;
  name?: string;
  query?: string;
  description?: string;
  observer_can_run?: boolean;
  discard_data?: boolean;
  frequency?: number;
  platform?: CommaSeparatedPlatformString;
  min_osquery_version?: string;
  automations_enabled?: boolean;
}

// response is ISchedulableQuery // better way to indicate this?

// Delete a query by name
/** DELETE /api/v1/fleet/queries/{name} */
export interface IDeleteQueryRequestBody {
  team_id?: number; // searches for a global query if omitted
}

// Delete a query by id
// DELETE /api/v1/fleet/queries/id/{id}
// (no body)

// Delete queries by id
/** POST /api/v1/fleet/queries/delete */
export interface IDeleteQueriesRequestBody {
  ids: number[];
}

export interface IDeleteQueriesResponse {
  deleted: number; // number of queries deleted
}

export interface IEditQueryFormFields {
  name: IFormField<string>;
  description: IFormField<string>;
  query: IFormField<string>;
  observer_can_run: IFormField<boolean>;
  discard_data: IFormField<boolean>;
  frequency: IFormField<number>;
  automations_enabled: IFormField<boolean>;
  platforms: IFormField<CommaSeparatedPlatformString>;
  min_osquery_version: IFormField<string>;
  logging: IFormField<QueryLoggingOption>;
}

export type QueryLoggingOption =
  | "snapshot"
  | "differential"
  | "differential_ignore_removals";
