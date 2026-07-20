import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import {
  ICreateSelfServiceCategoryFormData,
  IEditSelfServiceCategoryFormData,
  ISelfServiceCategory,
} from "interfaces/self_service_category";

const DEFAULT_TIMESTAMP = "2026-05-28T00:00:00Z";

export const createMockSelfServiceCategory = (
  overrides?: Partial<ISelfServiceCategory>
): ISelfServiceCategory => ({
  id: 1,
  name: "🌎 Browsers",
  fleet_id: 0,
  created_at: DEFAULT_TIMESTAMP,
  updated_at: DEFAULT_TIMESTAMP,
  ...overrides,
});

const categoriesUrl = baseUrl("/software/self_service_categories");
const categoryByIdUrl = baseUrl("/software/self_service_categories/:id");

// GET /software/self_service_categories?fleet_id=:id
export const listSelfServiceCategoriesHandler = (
  categories: Partial<ISelfServiceCategory>[] = [
    { id: 1, name: "🌎 Browsers" },
    { id: 2, name: "👬 Communication" },
    { id: 3, name: "🧰 Developer tools" },
    { id: 4, name: "💻 Productivity" },
    { id: 5, name: "🔐 Security" },
  ]
) =>
  http.get(categoriesUrl, () =>
    HttpResponse.json({
      self_service_categories: categories.map((c) =>
        createMockSelfServiceCategory(c)
      ),
    })
  );

// GET /device/:token/software/self_service_categories
// Device-token-scoped variant — BE derives the fleet from the token.
const deviceCategoriesUrl = baseUrl(
  "/device/:token/software/self_service_categories"
);

export const listDeviceSelfServiceCategoriesHandler = (
  categories: Partial<ISelfServiceCategory>[] = [
    { id: 1, name: "🌎 Browsers" },
    { id: 2, name: "👬 Communication" },
    { id: 3, name: "🧰 Developer tools" },
    { id: 4, name: "💻 Productivity" },
    { id: 5, name: "🔐 Security" },
  ]
) =>
  http.get(deviceCategoriesUrl, () =>
    HttpResponse.json({
      self_service_categories: categories.map((c) =>
        createMockSelfServiceCategory(c)
      ),
    })
  );

export const emptySelfServiceCategoriesHandler = http.get(categoriesUrl, () =>
  HttpResponse.json({ self_service_categories: [] })
);

export const emptyDeviceSelfServiceCategoriesHandler = http.get(
  deviceCategoriesUrl,
  () => HttpResponse.json({ self_service_categories: [] })
);

export const listSelfServiceCategoriesErrorHandler = http.get(
  categoriesUrl,
  () =>
    HttpResponse.json(
      { errors: [{ name: "base", reason: "Internal Server Error" }] },
      { status: 500 }
    )
);

// POST /software/self_service_categories
export const addSelfServiceCategoryHandler = http.post(
  categoriesUrl,
  async ({ request }) => {
    const body = (await request.json()) as ICreateSelfServiceCategoryFormData;
    return HttpResponse.json({
      self_service_category: createMockSelfServiceCategory({
        id: 99,
        name: body.name,
        fleet_id: body.fleet_id,
      }),
    });
  }
);

export const addSelfServiceCategoryConflictHandler = http.post(
  categoriesUrl,
  () =>
    HttpResponse.json(
      {
        errors: [
          {
            name: "name",
            reason:
              "A self-service category with this name already exists in this fleet.",
          },
        ],
      },
      { status: 409 }
    )
);

export const addSelfServiceCategoryErrorHandler = http.post(categoriesUrl, () =>
  HttpResponse.json(
    { errors: [{ name: "base", reason: "Internal Server Error" }] },
    { status: 500 }
  )
);

// PATCH /software/self_service_categories/:id
export const editSelfServiceCategoryHandler = http.patch(
  categoryByIdUrl,
  async ({ request, params }) => {
    const body = (await request.json()) as IEditSelfServiceCategoryFormData;
    return HttpResponse.json({
      self_service_category: createMockSelfServiceCategory({
        id: Number(params.id),
        name: body.name,
      }),
    });
  }
);

export const editSelfServiceCategoryConflictHandler = http.patch(
  categoryByIdUrl,
  () =>
    HttpResponse.json(
      {
        errors: [
          {
            name: "name",
            reason:
              "A self-service category with this name already exists in this fleet.",
          },
        ],
      },
      { status: 409 }
    )
);

export const editSelfServiceCategoryErrorHandler = http.patch(
  categoryByIdUrl,
  () =>
    HttpResponse.json(
      { errors: [{ name: "base", reason: "Internal Server Error" }] },
      { status: 500 }
    )
);

// DELETE /software/self_service_categories/:id
export const deleteSelfServiceCategoryHandler = http.delete(
  categoryByIdUrl,
  () => new HttpResponse(null, { status: 204 })
);

export const deleteSelfServiceCategoryErrorHandler = http.delete(
  categoryByIdUrl,
  () =>
    HttpResponse.json(
      { errors: [{ name: "base", reason: "Internal Server Error" }] },
      { status: 500 }
    )
);
