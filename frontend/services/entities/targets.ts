/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { ITargetsAPIResponse, ISelectedTargets } from "interfaces/target";

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
};
