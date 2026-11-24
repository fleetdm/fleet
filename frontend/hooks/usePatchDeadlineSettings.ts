import { useQuery, UseQueryOptions, UseQueryResult } from "react-query";

import { IPatchDeadlineSettings } from "interfaces/patch_deadline_settings";
import patchDeadlineSettingsAPI from "services/entities/patch_deadline_settings";

/** Fetches a fleet's patch deadline settings via React Query. Shared cache key
 * means multiple consumers on the same page (banner + Library settings) dedupe
 * through React Query instead of double-fetching. Stays idle while `fleetId`
 * is undefined. */
const usePatchDeadlineSettings = (
  fleetId: number | undefined,
  options?: Omit<
    UseQueryOptions<IPatchDeadlineSettings, Error>,
    "queryKey" | "queryFn"
  >
): UseQueryResult<IPatchDeadlineSettings, Error> =>
  useQuery<IPatchDeadlineSettings, Error>(
    ["patchDeadlineSettings", fleetId],
    () => patchDeadlineSettingsAPI.getSettings(fleetId as number),
    {
      refetchOnWindowFocus: false,
      ...options,
      enabled: fleetId !== undefined && (options?.enabled ?? true),
    }
  );

export default usePatchDeadlineSettings;
