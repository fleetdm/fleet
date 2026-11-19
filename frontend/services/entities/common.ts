/** Maximum per_page the server accepts; larger values are clamped to this. */
export const MAX_SERVER_PER_PAGE = 1000;

/**
 * Fetches every page of a paginated list endpoint, concatenating the items.
 * `fetchPage` is called with a 0-indexed page and the page size; the loop
 * stops when a page returns fewer than `perPage` items. Use for endpoints
 * whose responses carry no pagination metadata; prefer looping on
 * `meta.has_next_results` when the response includes it.
 */
export const fetchAllPages = async <T>(
  fetchPage: (page: number, perPage: number) => Promise<T[]>,
  perPage: number = MAX_SERVER_PER_PAGE
): Promise<T[]> => {
  const all: T[] = [];
  let page = 0;
  let lastPageLength = perPage;
  while (lastPageLength === perPage) {
    // pages must be fetched sequentially
    // eslint-disable-next-line no-await-in-loop
    const items = await fetchPage(page, perPage);
    all.push(...items);
    lastPageLength = items.length;
    page += 1;
  }
  return all;
};

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
