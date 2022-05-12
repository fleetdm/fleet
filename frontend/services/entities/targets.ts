/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import sendMockRequest from "services/mock_service";
import endpoints from "utilities/endpoints";
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

export interface ITargetsCount {
  targets_count: number;
  targets_online: number;
  targets_offline: number;
  targets_missing_in_action: number;
}
// // TODO: deprecated until frontend\components\forms\fields\SelectTargetsDropdown
// // is fully replaced with frontend\components\TargetsInput
// const DEPRECATED_defaultSelected = {
//   hosts: [],
//   labels: [],
// };

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

  labels: () => {
    const { LABELS } = endpoints;

    return sendMockRequest("GET", `${LABELS}?count=false`);
  },

  search: (query: string) => {
    const { TARGETS } = endpoints;
    return sendMockRequest("GET", `${TARGETS}?query=${query}`);
  },

  count: (selected: ISelectedTargets | null): Promise<ITargetsCount> => {
    if (!selected) {
      return Promise.reject("Invalid usage: no selected targets");
    }
    const { TARGETS } = endpoints;
    console.log("selected", selected);

    return sendMockRequest("POST", `${TARGETS}/count`, selected);
  },
  // TODO: deprecated until frontend\components\forms\fields\SelectTargetsDropdown
  // is fully replaced with frontend\components\TargetsInput
  // DEPRECATED_loadAll: (
  //   query = "",
  //   queryId = null,
  //   selected = DEPRECATED_defaultSelected
  // ) => {
  //   const { TARGETS } = endpoints;

  //   return sendRequest("POST", TARGETS, {
  //     query,
  //     query_id: queryId,
  //     selected,
  //   }).then((response) => {
  //     const { targets } = response;

  //     return {
  //       ...response,
  //       targets: [
  //         ...appendTargetTypeToTargets(targets.hosts, "hosts"),
  //         ...appendTargetTypeToTargets(targets.labels, "labels"),
  //         ...appendTargetTypeToTargets(targets.teams, "teams"),
  //       ],
  //     };
  //   });
  // },
};
