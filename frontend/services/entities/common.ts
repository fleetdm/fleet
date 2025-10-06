// TODO - apply broadly
export interface ListEntitiesResponsePaginationCommon {
  has_next_results: boolean;
  has_previous_results: boolean;
}

export interface ListEntitiesResponseCommon {
  meta: ListEntitiesResponsePaginationCommon;
  count: number;
}

export type OrderDirection = "asc" | "desc";

export interface PaginationParams {
  page: number;
  per_page: number;
}
