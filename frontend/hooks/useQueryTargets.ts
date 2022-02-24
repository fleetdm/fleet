import { useQuery, UseQueryResult } from "react-query";
import { filter } from "lodash";
import { v4 as uuidv4 } from "uuid";

import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { ISelectedTargets } from "interfaces/target";
import targetsAPI from "services/entities/targets";

export interface ITargetsLabels {
  allHostsLabels?: ILabel[];
  platformLabels?: ILabel[];
  otherLabels?: ILabel[];
  teams?: ITeam[];
  labelCount?: number;
}

export interface ITargetsQueryResponse extends ITargetsLabels {
  targetsTotalCount: number;
  targetsOnlinePercent: number;
  relatedHosts?: IHost[];
}

export interface ITargetsQueryKey {
  scope: string;
  query: string;
  queryId: number | null;
  selected: ISelectedTargets;
  includeLabels: boolean;
}

const STALE_TIME = 60000;

const getTargets = async (
  queryKey: ITargetsQueryKey
): Promise<ITargetsQueryResponse> => {
  const { query, queryId, selected, includeLabels } = queryKey;

  try {
    const {
      targets,
      targets_count: targetsTotalCount,
      targets_online: targetsOnline,
    } = await targetsAPI.loadAll({
      query,
      queryId,
      selected,
    });
    let responseLabels: ITargetsLabels = {};

    if (includeLabels) {
      const { labels } = targets;

      const all = filter(
        labels,
        ({ display_text: text }) => text === "All Hosts"
      ).map((label) => ({ ...label, uuid: uuidv4() }));

      const platforms = filter(
        labels,
        ({ display_text: text }) =>
          text === "macOS" || text === "MS Windows" || text === "All Linux"
      ).map((label) => ({ ...label, uuid: uuidv4() }));

      const other = filter(
        labels,
        ({ label_type: type }) => type === "regular"
      ).map((label) => ({ ...label, uuid: uuidv4() }));

      const teams = targets.teams.map((team) => ({ ...team, uuid: uuidv4() }));

      const labelCount =
        all.length + platforms.length + other.length + teams.length;

      responseLabels = {
        allHostsLabels: all,
        platformLabels: platforms,
        otherLabels: other,
        teams,
        labelCount,
      };
    }

    const targetsOnlinePercent =
      targetsTotalCount > 0
        ? Math.round((targetsOnline / targetsTotalCount) * 100)
        : 0;

    return Promise.resolve({
      ...responseLabels,
      relatedHosts: query ? [...targets.hosts] : [],
      targetsTotalCount,
      targetsOnlinePercent,
    });
  } catch (err) {
    return Promise.reject(err);
  }
};

export const useQueryTargets = (
  targetsQueryKey: ITargetsQueryKey[],
  options: { onSuccess: (data: ITargetsQueryResponse) => void }
): UseQueryResult<ITargetsQueryResponse, Error> => {
  return useQuery<
    ITargetsQueryResponse,
    Error,
    ITargetsQueryResponse,
    ITargetsQueryKey[]
  >(
    targetsQueryKey,
    ({ queryKey }) => {
      return getTargets(queryKey[0]);
    },
    {
      refetchOnWindowFocus: false,
      staleTime: STALE_TIME,
      onSuccess: options.onSuccess,
    }
  );
};

export default useQueryTargets;
