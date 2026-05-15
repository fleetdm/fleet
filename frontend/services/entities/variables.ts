import { IVariable, IVariablePayload } from "interfaces/variables";
import sendRequest from "services";
import { buildQueryStringFromParams } from "utilities/url";
import endpoints from "utilities/endpoints";

export interface IListVariablesRequestApiParams {
  page?: number;
  per_page?: number;
}

export interface IListVariablesResponse {
  custom_variables: IVariable[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
  count: number;
}

export default {
  getVariables(
    params: IListVariablesRequestApiParams
  ): Promise<IListVariablesResponse> {
    const { VARIABLES } = endpoints;
    const path = `${VARIABLES}?${buildQueryStringFromParams({
      page: params.page,
      per_page: params.per_page,
    })}`;

    return sendRequest("GET", path);
  },

  addVariable(variable: IVariablePayload) {
    const { VARIABLES } = endpoints;
    return sendRequest("POST", VARIABLES, variable);
  },

  deleteVariable(variableId: number) {
    const { VARIABLES } = endpoints;
    return sendRequest("DELETE", `${VARIABLES}/${variableId}`);
  },
};
