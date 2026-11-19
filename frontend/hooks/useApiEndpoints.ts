import { useQuery, UseQueryOptions, UseQueryResult } from "react-query";

import { IApiEndpoint } from "interfaces/api_endpoint";
import apiEndpointsAPI from "services/entities/api_endpoints";

/** Fetches the catalog of endpoints an API-only user can be granted. Consumers
 * share one cache key, so React Query dedupes the request across them. */
const useApiEndpoints = (
  options?: Omit<UseQueryOptions<IApiEndpoint[], Error>, "queryKey" | "queryFn">
): UseQueryResult<IApiEndpoint[], Error> =>
  useQuery<IApiEndpoint[], Error>(
    ["api_endpoints"],
    () => apiEndpointsAPI.loadAll(),
    {
      refetchOnWindowFocus: false,
      ...options,
    }
  );

export default useApiEndpoints;
