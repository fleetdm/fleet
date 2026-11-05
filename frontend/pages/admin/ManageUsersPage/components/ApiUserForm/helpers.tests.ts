import { IApiEndpointRef } from "interfaces/api_endpoint";

import {
  ApiUserFormState,
  dropUnlistedEndpoints,
  validateApiUserForm,
} from "./helpers";

const LIST_HOSTS: IApiEndpointRef = {
  method: "GET",
  path: "/api/v1/fleet/hosts",
};

const LIST_USERS: IApiEndpointRef = {
  method: "GET",
  path: "/api/v1/fleet/users",
};

const REMOVED_ROUTE: IApiEndpointRef = {
  method: "GET",
  path: "/api/v1/fleet/mdm/profiles",
};

const CATALOG: IApiEndpointRef[] = [LIST_HOSTS, LIST_USERS];

const SPECIFIC_ENDPOINTS: ApiUserFormState = {
  name: "CI token",
  global_role: "observer",
  isGlobalUser: true,
  fleets: [],
  isSpecificEndpoints: true,
  api_endpoints: [LIST_HOSTS],
};

describe("dropUnlistedEndpoints", () => {
  it("drops endpoints the catalog doesn't list", () => {
    expect(
      dropUnlistedEndpoints([LIST_HOSTS, REMOVED_ROUTE], CATALOG)
    ).toEqual([LIST_HOSTS]);
  });

  it("keeps endpoints the catalog lists under a different spelling", () => {
    const hostDetails = { method: "GET", path: "/api/v1/fleet/hosts/:id" };
    const sameAsServerMatches = [
      { method: "get", path: LIST_HOSTS.path },
      { method: "GET", path: "/api/v1/fleet/hosts/:host_id" },
      { method: "GET", path: "/api/v1/fleet/hosts/{id}" },
    ];

    expect(
      dropUnlistedEndpoints(sameAsServerMatches, [...CATALOG, hostDetails])
    ).toEqual(sameAsServerMatches);
  });

  it("matches on method and path together", () => {
    const otherMethodSamePath = { method: "POST", path: LIST_HOSTS.path };

    expect(dropUnlistedEndpoints([otherMethodSamePath], CATALOG)).toEqual([]);
  });

  it("keeps every endpoint when the catalog hasn't loaded", () => {
    const selected = [LIST_HOSTS, REMOVED_ROUTE];

    expect(dropUnlistedEndpoints(selected, undefined)).toEqual(selected);
    expect(dropUnlistedEndpoints(selected, [])).toEqual(selected);
  });
});

describe("validateApiUserForm", () => {
  it("accepts a selection the catalog lists", () => {
    expect(
      validateApiUserForm(SPECIFIC_ENDPOINTS, {
        isPremiumTier: true,
        apiEndpointCatalog: CATALOG,
      })
    ).toEqual({});
  });

  it("requires an endpoint when only unlisted endpoints are selected", () => {
    const errors = validateApiUserForm(
      { ...SPECIFIC_ENDPOINTS, api_endpoints: [REMOVED_ROUTE] },
      { isPremiumTier: true, apiEndpointCatalog: CATALOG }
    );

    expect(errors.api_endpoints).toBe("Select at least one API endpoint");
  });

  it("doesn't flag unlisted endpoints when the catalog hasn't loaded", () => {
    const errors = validateApiUserForm(
      { ...SPECIFIC_ENDPOINTS, api_endpoints: [REMOVED_ROUTE] },
      { isPremiumTier: true }
    );

    expect(errors.api_endpoints).toBeUndefined();
  });

  it("skips the endpoint rule on free tier", () => {
    const errors = validateApiUserForm(
      { ...SPECIFIC_ENDPOINTS, api_endpoints: [REMOVED_ROUTE] },
      { isPremiumTier: false, apiEndpointCatalog: CATALOG }
    );

    expect(errors.api_endpoints).toBeUndefined();
  });
});
