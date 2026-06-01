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
