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

// -----------------------------------------------------------------------------
// TEMP DEV MOCKS — BE for #39018 self-service categories is not yet implemented.
// This block is stripped from production builds by webpack DCE on the
// `process.env.NODE_ENV === "development"` guard. Delete the block (and the
// `if (USE_FE_MOCKS)` short-circuits below) once the real routes ship.
// -----------------------------------------------------------------------------
const USE_FE_MOCKS = process.env.NODE_ENV === "development";
const MOCK_LATENCY_MS = 200;
const delay = <T>(value: T): Promise<T> =>
  new Promise((resolve) => setTimeout(() => resolve(value), MOCK_LATENCY_MS));

const nowIso = () => new Date().toISOString();
const mockStore: ISelfServiceCategory[] = [
  {
    id: 1,
    name: "🌎 Browsers",
    fleet_id: 0,
    created_at: nowIso(),
    updated_at: nowIso(),
  },
  {
    id: 2,
    name: "👬 Communication",
    fleet_id: 0,
    created_at: nowIso(),
    updated_at: nowIso(),
  },
  {
    id: 3,
    name: "🧰 Developer tools",
    fleet_id: 0,
    created_at: nowIso(),
    updated_at: nowIso(),
  },
  {
    id: 4,
    name: "💻 Productivity",
    fleet_id: 0,
    created_at: nowIso(),
    updated_at: nowIso(),
  },
  {
    id: 5,
    name: "🔐 Security",
    fleet_id: 0,
    created_at: nowIso(),
    updated_at: nowIso(),
  },
];
let mockNextId = 100;

// Ignore fleetId in the mock list so any fleet the user picks shows seeded
// data — the BE will scope by fleet for real.
const mockList = (): Promise<ISelfServiceCategoriesResponse> =>
  delay({ self_service_categories: [...mockStore] });

const mockAdd = (
  formData: ICreateSelfServiceCategoryFormData
): Promise<ISelfServiceCategoryResponse> => {
  const conflict = mockStore.find(
    (c) =>
      c.fleet_id === formData.fleet_id &&
      c.name.toLowerCase() === formData.name.toLowerCase()
  );
  if (conflict) {
    return Promise.reject({ status: 409 });
  }
  mockNextId += 1;
  const newCategory: ISelfServiceCategory = {
    id: mockNextId,
    name: formData.name,
    fleet_id: formData.fleet_id,
    created_at: nowIso(),
    updated_at: nowIso(),
  };
  mockStore.push(newCategory);
  return delay({ self_service_category: newCategory });
};

const mockEdit = (
  id: number,
  formData: IEditSelfServiceCategoryFormData
): Promise<ISelfServiceCategoryResponse> => {
  const idx = mockStore.findIndex((c) => c.id === id);
  if (idx === -1) return Promise.reject({ status: 404 });
  const conflict = mockStore.find(
    (c) =>
      c.id !== id &&
      c.fleet_id === mockStore[idx].fleet_id &&
      c.name.toLowerCase() === formData.name.toLowerCase()
  );
  if (conflict) return Promise.reject({ status: 409 });
  mockStore[idx] = {
    ...mockStore[idx],
    name: formData.name,
    updated_at: nowIso(),
  };
  return delay({ self_service_category: mockStore[idx] });
};

const mockDestroy = (id: number): Promise<void> => {
  const idx = mockStore.findIndex((c) => c.id === id);
  if (idx !== -1) mockStore.splice(idx, 1);
  return delay(undefined);
};
// -----------------------------------------------------------------------------
// END TEMP DEV MOCKS
// -----------------------------------------------------------------------------

export default {
  list: (fleetId: number): Promise<ISelfServiceCategoriesResponse> => {
    if (USE_FE_MOCKS) return mockList();
    const { SELF_SERVICE_CATEGORIES } = endpoints;
    const queryString = buildQueryStringFromParams({ fleet_id: fleetId });
    return sendRequest("GET", `${SELF_SERVICE_CATEGORIES}?${queryString}`);
  },

  add: (
    formData: ICreateSelfServiceCategoryFormData
  ): Promise<ISelfServiceCategoryResponse> => {
    if (USE_FE_MOCKS) return mockAdd(formData);
    const { SELF_SERVICE_CATEGORIES } = endpoints;
    return sendRequest("POST", SELF_SERVICE_CATEGORIES, formData);
  },

  edit: (
    id: number,
    formData: IEditSelfServiceCategoryFormData
  ): Promise<ISelfServiceCategoryResponse> => {
    if (USE_FE_MOCKS) return mockEdit(id, formData);
    const { SELF_SERVICE_CATEGORY } = endpoints;
    return sendRequest("PATCH", SELF_SERVICE_CATEGORY(id), formData);
  },

  destroy: (id: number) => {
    if (USE_FE_MOCKS) return mockDestroy(id);
    const { SELF_SERVICE_CATEGORY } = endpoints;
    return sendRequest("DELETE", SELF_SERVICE_CATEGORY(id));
  },
};
