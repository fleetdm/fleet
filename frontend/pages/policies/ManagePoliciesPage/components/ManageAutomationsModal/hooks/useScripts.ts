import { useQuery } from "react-query";
import { omit } from "lodash";

import scriptsAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";

const SCRIPTS_PAGE_SIZE = 1000;

interface IUseScriptsArgs {
  teamId: number;
  enabled: boolean;
}

const useScripts = ({ teamId, enabled }: IUseScriptsArgs) =>
  useQuery<IScriptsResponse, Error, IScriptsResponse, [IListScriptsQueryKey]>(
    [
      {
        scope: "scripts",
        page: 0,
        per_page: SCRIPTS_PAGE_SIZE,
        fleet_id: teamId,
      },
    ],
    ({ queryKey: [key] }) => scriptsAPI.getScripts(omit(key, "scope")),
    { enabled, staleTime: 30_000 }
  );

export default useScripts;
