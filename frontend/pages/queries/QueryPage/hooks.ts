import { useQuery, UseQueryResult } from "react-query";
import { filter } from "lodash";

import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { ISelectedTargets } from "interfaces/target";
import targetsAPI from "services/entities/targets";

export interface ITargetsQueryResponse {
  targetsTotalCount: number;
  targetsOnlinePercent: number;
  allHostsLabel?: ILabel[];
  platformLabels?: ILabel[];
  otherLabels?: ILabel[];
  teams?: ITeam[];
  labelCount?: number;
  relatedHosts?: IHost[];
}

export interface ITargetsQueryKey {
  scope: string;
  query: string;
  queryId: number | null;
  selected: ISelectedTargets;
  includeLabels: boolean;
}

const getTargets = async (
  queryKey: ITargetsQueryKey
): Promise<ITargetsQueryResponse> => {
  const { query, queryId, selected, includeLabels } = queryKey;

  const {
    targets,
    targets_count: targetsTotalCount,
    targets_online: targetsOnline,
  } = await targetsAPI.loadAll({
    query,
    queryId,
    selected,
  });

  let response: Partial<ITargetsQueryResponse> = {};

  if (includeLabels) {
    const { labels } = targets;

    const all = filter(
      labels,
      ({ display_text: text }) => text === "All Hosts"
    );

    const platforms = filter(
      labels,
      ({ display_text: text }) =>
        text === "macOS" || text === "MS Windows" || text === "All Linux"
    );

    const other = filter(labels, ({ label_type: type }) => type === "regular");

    const labelCount =
      all.length + platforms.length + other.length + targets.teams.length;

    response = {
      allHostsLabel: all,
      platformLabels: platforms,
      otherLabels: other,
      teams: targets.teams,
      labelCount,
    };
  }

  const targetsOnlinePercent =
    targetsTotalCount > 0
      ? Math.round((targetsOnline / targetsTotalCount) * 100)
      : 0;

  return Promise.resolve({
    ...response,
    relatedHosts: query ? [...targets.hosts] : [],
    targetsTotalCount,
    targetsOnlinePercent,
  });
};

export const useTargetsQuery = (
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
      staleTime: 15000,
      onSuccess: options.onSuccess,
    }
  );
};

export default useTargetsQuery;
