import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";
import {
  ICreateSelfServiceCategoryFormData,
  IEditSelfServiceCategoryFormData,
  ISelfServiceCategory,
} from "interfaces/self_service_category";

export interface ISelfServiceCategoriesResponse {
  self_service_categories: ISelfServiceCategory[];
}

export interface ISelfServiceCategoryResponse {
  self_service_category: ISelfServiceCategory;
}

export default {
  getCategories: (fleetId: number): Promise<ISelfServiceCategoriesResponse> => {
    const { SELF_SERVICE_CATEGORIES } = endpoints;
    const queryString = buildQueryStringFromParams({ fleet_id: fleetId });
    return sendRequest("GET", `${SELF_SERVICE_CATEGORIES}?${queryString}`);
  },

  // Device-token-scoped variant — the BE derives the fleet from the device
  // token so end users see the categories defined for their own fleet rather
  // than the global (fleet_id=0) set.
  getDeviceCategories: (
    deviceToken: string
  ): Promise<ISelfServiceCategoriesResponse> => {
    const { DEVICE_SELF_SERVICE_CATEGORIES } = endpoints;
    return sendRequest("GET", DEVICE_SELF_SERVICE_CATEGORIES(deviceToken));
  },

  addCategory: (
    formData: ICreateSelfServiceCategoryFormData
  ): Promise<ISelfServiceCategoryResponse> => {
    const { SELF_SERVICE_CATEGORIES } = endpoints;
    return sendRequest("POST", SELF_SERVICE_CATEGORIES, formData);
  },

  updateCategory: (
    id: number,
    formData: IEditSelfServiceCategoryFormData
  ): Promise<ISelfServiceCategoryResponse> => {
    const { SELF_SERVICE_CATEGORY } = endpoints;
    return sendRequest("PATCH", SELF_SERVICE_CATEGORY(id), formData);
  },

  deleteCategory: (id: number) => {
    const { SELF_SERVICE_CATEGORY } = endpoints;
    return sendRequest("DELETE", SELF_SERVICE_CATEGORY(id));
  },
};
