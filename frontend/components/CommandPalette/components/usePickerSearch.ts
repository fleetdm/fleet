import { useEffect, useState } from "react";
import { useQuery, UseQueryOptions } from "react-query";

const DEBOUNCE_MS = 200;

interface IUsePickerSearchOptions<TResponse, TItem> {
  /** Raw search input from the parent. Debounced internally. */
  search: string;
  /**
   * React Query key prefix WITHOUT the search term — e.g.,
   * `["commandPaletteHosts", teamId]`. The hook appends the debounced
   * query internally so the cache key stays in lockstep with the value
   * passed into queryFn (using raw search here would tag fresh cache
   * entries with stale data while the debounce settles).
   *
   * Must be an array — react-query accepts a bare string as a QueryKey,
   * but spreading one into the cache key would iterate its characters.
   */
  queryKeyPrefix: readonly unknown[];
  /** Function that fetches the response given the debounced query. */
  queryFn: (debouncedQuery: string) => Promise<TResponse>;
  /** Extract the displayed item array from the response. */
  selectItems: (data: TResponse | undefined) => TItem[];
  /** Optional overrides for react-query (rarely needed). */
  queryOptions?: Omit<
    UseQueryOptions<TResponse, Error>,
    "queryKey" | "queryFn"
  >;
}

/**
 * Shared scaffolding for the command-palette pickers: a 200ms debounce on
 * the raw `search` input + a `useQuery` against a server endpoint that
 * pre-filters by the debounced query. Returns the extracted item array,
 * the loading flag, and the resolved debounced query (used for empty-state
 * copy in the consumer).
 */
const usePickerSearch = <TResponse, TItem>({
  search,
  queryKeyPrefix,
  queryFn,
  selectItems,
  queryOptions,
}: IUsePickerSearchOptions<TResponse, TItem>) => {
  const [debouncedQuery, setDebouncedQuery] = useState(search.trim());

  useEffect(() => {
    const id = window.setTimeout(() => {
      setDebouncedQuery(search.trim());
    }, DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [search]);

  const { data, isLoading } = useQuery<TResponse, Error>(
    [...queryKeyPrefix, debouncedQuery],
    () => queryFn(debouncedQuery),
    {
      keepPreviousData: true,
      staleTime: 30000,
      // Pickers are short-lived UI; release cached entries 60s after the
      // last consumer unmounts so a long session doesn't accumulate
      // distinct (team, query) tuples indefinitely (react-query's
      // default cacheTime is 5 min).
      cacheTime: 60000,
      ...queryOptions,
    }
  );

  return {
    items: selectItems(data),
    isLoading,
    debouncedQuery,
  };
};

export default usePickerSearch;
