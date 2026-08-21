import { omit } from "lodash";
import { useQuery } from "react-query";
import mdmApi, {
  IGetProfilesApiParams,
  IMdmProfilesResponse,
} from "services/entities/mdm";

const PROFILES_PAGE_SIZE = 1000;

interface IUseProfilesArgs {
  fleetId: number;
  enabled: boolean;
}

type IUseProfilesQueryKey = IGetProfilesApiParams & { scope: "profiles" };

const useProfiles = ({ fleetId, enabled }: IUseProfilesArgs) =>
  useQuery<
    IMdmProfilesResponse,
    Error,
    IMdmProfilesResponse,
    [IUseProfilesQueryKey]
  >(
    [
      {
        scope: "profiles",
        page: 0,
        per_page: PROFILES_PAGE_SIZE,
        fleet_id: fleetId,
      },
    ],
    ({ queryKey: [key] }) => mdmApi.getProfiles(omit(key, "scope")),
    { enabled, staleTime: 30_000 }
  );

export default useProfiles;
