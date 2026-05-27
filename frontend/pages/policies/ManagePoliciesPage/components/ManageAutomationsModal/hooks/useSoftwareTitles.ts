import { useQuery } from "react-query";
import { omit } from "lodash";

import { CommaSeparatedPlatformString } from "interfaces/platform";
import softwareAPI, {
  ISoftwareTitlesQueryKey,
  ISoftwareTitlesResponse,
} from "services/entities/software";

const SOFTWARE_PAGE_SIZE = 1000;

interface IUseSoftwareTitlesArgs {
  teamId: number;
  enabled: boolean;
}

const useSoftwareTitles = ({ teamId, enabled }: IUseSoftwareTitlesArgs) =>
  useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: 0,
        perPage: SOFTWARE_PAGE_SIZE,
        query: "",
        orderDirection: "desc",
        orderKey: "hosts_count",
        teamId,
        availableForInstall: true,
        platform: "darwin,windows,linux" as CommaSeparatedPlatformString,
      },
    ],
    ({ queryKey: [key] }) => softwareAPI.getSoftwareTitles(omit(key, "scope")),
    { enabled, staleTime: 30_000 }
  );

export default useSoftwareTitles;
