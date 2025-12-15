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

export const DEFAULT_PAGINATION_PARAMS: PaginationParams = {
  page: 1,
  per_page: 10,
};
