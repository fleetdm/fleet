import { IVariable, IVariableFormData } from "interfaces/variables";
import sendRequest from "services";
import { buildQueryStringFromParams } from "utilities/url";
import endpoints from "utilities/endpoints";

export interface IListVariablesApiParams {
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
    params: IListVariablesApiParams
  ): Promise<IListVariablesResponse> {
    const { GLOBAL_VARIABLES } = endpoints;
    const path = `${GLOBAL_VARIABLES}?${buildQueryStringFromParams({
      page: params.page,
      per_page: params.per_page,
    })}`;

    return sendRequest("GET", path);
  },

  addVariable(variable: IVariableFormData) {
    const { GLOBAL_VARIABLES } = endpoints;
    return sendRequest("POST", GLOBAL_VARIABLES, variable);
  },

  deleteVariable(variableId: number) {
    const { GLOBAL_VARIABLES } = endpoints;
    return sendRequest("DELETE", `${GLOBAL_VARIABLES}/${variableId}`);
  },
};
