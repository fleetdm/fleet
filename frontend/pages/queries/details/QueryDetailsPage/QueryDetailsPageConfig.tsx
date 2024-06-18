// TODO
export const QUERY_DETAILS_PAGE_FILTER_KEYS = ["model", "vendor"] as const;

// TODO: refactor to use this type as the location.query prop of the page
export type QueryDetailsPageQueryParams = Record<
  | "order_key"
  | "order_direction"
  | typeof QUERY_DETAILS_PAGE_FILTER_KEYS[number],
  string
>;

export const DEFAULT_SORT_HEADER = "host_name";
export const DEFAULT_SORT_DIRECTION = "asc";
