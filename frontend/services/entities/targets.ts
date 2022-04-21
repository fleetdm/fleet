/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { ITargetsAPIResponse, ISelectedTargets } from "interfaces/target";
import appendTargetTypeToTargets from "utilities/append_target_type_to_targets";

interface ITargetsProps {
  query?: string;
  queryId?: number | null;
  selected: ISelectedTargets;
}

const defaultSelected = {
  hosts: [],
  labels: [],
  teams: [],
};

// TODO: deprecated until frontend\components\forms\fields\SelectTargetsDropdown
// is fully replaced with frontend\components\TargetsInput
const DEPRECATED_defaultSelected = {
  hosts: [],
  labels: [],
};

export default {
  loadAll: ({
    query = "",
    queryId = null,
    selected = defaultSelected,
  }: ITargetsProps): Promise<ITargetsAPIResponse> => {
    const { TARGETS } = endpoints;

    return sendRequest("POST", TARGETS, {
      query,
      query_id: queryId,
      selected,
    });
  },
  // TODO: deprecated until frontend\components\forms\fields\SelectTargetsDropdown
  // is fully replaced with frontend\components\TargetsInput
  DEPRECATED_loadAll: (
    query = "",
    queryId = null,
    selected = DEPRECATED_defaultSelected
  ) => {
    const { TARGETS } = endpoints;

    return sendRequest("POST", TARGETS, {
      query,
      query_id: queryId,
      selected,
    }).then((response) => {
      const { targets } = response;

      return {
        ...response,
        targets: [
          ...appendTargetTypeToTargets(targets.hosts, "hosts"),
          ...appendTargetTypeToTargets(targets.labels, "labels"),
          ...appendTargetTypeToTargets(targets.teams, "teams"),
        ],
      };
    });
  },
};
