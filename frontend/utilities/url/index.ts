import { isEmpty, reduce, omitBy, Dictionary } from "lodash";

type QueryValues = string | number | boolean | undefined | null;
export type QueryParams = Record<string, QueryValues>;
type FilteredQueryValues = string | number | boolean;
type FilteredQueryParams = Record<string, FilteredQueryValues>;

const reduceQueryParams = (
  params: string[],
  value: FilteredQueryValues,
  key: string
) => {
  key && params.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`);
  return params;
};

const filterEmptyParams = (queryParams: QueryParams) => {
  return omitBy(
    queryParams,
    (value) => value === undefined || value === "" || value === null
  ) as Dictionary<FilteredQueryValues>;
};

/**
 * creates a query string from a query params object. If a value is undefined, null,
 * or an empty string on the queryParams object, that key-value pair will be
 * excluded from the query string.
 */
export const buildQueryStringFromParams = (queryParams: QueryParams) => {
  const filteredParams = filterEmptyParams(queryParams);

  let queryString = "";
  if (!isEmpty(queryParams)) {
    queryString = reduce<FilteredQueryParams, string[]>(
      filteredParams,
      reduceQueryParams,
      []
    ).join("&");
  }
  return queryString;
};
