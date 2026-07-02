import {
  ICustomHostVital,
  ICustomHostVitalFormData,
} from "interfaces/custom_host_vitals";
import sendRequest from "services";
import { buildQueryStringFromParams } from "utilities/url";
import endpoints from "utilities/endpoints";

export interface IListCustomHostVitalsApiParams {
  page?: number;
  per_page?: number;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
}

export interface IListCustomHostVitalsResponse {
  custom_host_vitals: ICustomHostVital[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
  count: number;
}

export default {
  getCustomHostVitals(
    params: IListCustomHostVitalsApiParams
  ): Promise<IListCustomHostVitalsResponse> {
    const { CUSTOM_HOST_VITALS } = endpoints;
    const path = `${CUSTOM_HOST_VITALS}?${buildQueryStringFromParams({
      page: params.page,
      per_page: params.per_page,
      query: params.query,
      order_key: params.order_key,
      order_direction: params.order_direction,
    })}`;

    return sendRequest("GET", path);
  },

  addCustomHostVital(vital: ICustomHostVitalFormData) {
    const { CUSTOM_HOST_VITALS } = endpoints;
    return sendRequest("POST", CUSTOM_HOST_VITALS, vital);
  },

  updateCustomHostVital(id: number, vital: ICustomHostVitalFormData) {
    const { CUSTOM_HOST_VITALS } = endpoints;
    return sendRequest("PATCH", `${CUSTOM_HOST_VITALS}/${id}`, vital);
  },

  deleteCustomHostVital(id: number) {
    const { CUSTOM_HOST_VITALS } = endpoints;
    return sendRequest("DELETE", `${CUSTOM_HOST_VITALS}/${id}`);
  },
};
