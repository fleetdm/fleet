import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  IBaselinesResponse,
  IApplyBaselineRequest,
  IApplyBaselineResponse,
} from "interfaces/baseline";

export default {
  loadAll: (): Promise<IBaselinesResponse> => {
    const { MDM_BASELINES } = endpoints;
    return sendRequest("GET", MDM_BASELINES);
  },

  apply: (data: IApplyBaselineRequest): Promise<IApplyBaselineResponse> => {
    const { MDM_BASELINES_APPLY } = endpoints;
    return sendRequest("POST", MDM_BASELINES_APPLY, data);
  },

  remove: (baselineId: string, teamId: number): Promise<void> => {
    const path = endpoints.MDM_BASELINE_REMOVE(baselineId, teamId);
    return sendRequest("DELETE", path);
  },
};
