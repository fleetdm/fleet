import { useCallback, useContext } from "react";
import { useQueryClient } from "react-query";

import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";

/**
 * Single setter for the app config. Updates the AppContext AND the React
 * Query cache in one call, so pages that write config don't have to refetch
 * (which is unsafe under read-replica lag).
 *
 * Call this with the response from a config write, e.g.:
 *   const updateAppConfig = useUpdateAppConfig();
 *   const updated = await configAPI.update(diff);
 *   updateAppConfig(updated);
 */
const useUpdateAppConfig = () => {
  const { setConfig } = useContext(AppContext);
  const queryClient = useQueryClient();

  return useCallback(
    (config: IConfig) => {
      setConfig(config);
      queryClient.setQueryData(["config"], config);
    },
    [setConfig, queryClient]
  );
};

export default useUpdateAppConfig;
