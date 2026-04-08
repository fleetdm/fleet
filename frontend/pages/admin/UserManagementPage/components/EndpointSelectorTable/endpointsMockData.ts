export interface IApiEndpoint {
  id: string;
  name: string;
  method: "GET" | "POST" | "PATCH" | "DELETE" | "PUT";
  path: string;
  deprecated?: boolean;
}

// Mock data representing Fleet API endpoints.
// TODO: Replace with real API call once the backend endpoint listing is available.
const FLEET_API_ENDPOINTS: IApiEndpoint[] = [
  {
    id: "list-hosts",
    name: "List hosts",
    method: "GET",
    path: "/api/v1/fleet/hosts",
  },
  {
    id: "get-host",
    name: "Get host",
    method: "GET",
    path: "/api/v1/fleet/hosts/:id",
  },
  {
    id: "delete-host",
    name: "Delete host",
    method: "DELETE",
    path: "/api/v1/fleet/hosts/:id",
  },
  {
    id: "batch-delete-hosts",
    name: "Batch-delete hosts",
    method: "POST",
    path: "/api/v1/fleet/hosts/delete",
  },
  {
    id: "get-host-certificates",
    name: "Get host's certificates",
    method: "GET",
    path: "/api/v1/fleet/hosts/:id/certificates",
  },
  {
    id: "list-labels",
    name: "List labels",
    method: "GET",
    path: "/api/v1/fleet/labels",
  },
  {
    id: "create-label",
    name: "Create label",
    method: "POST",
    path: "/api/v1/fleet/labels",
  },
  {
    id: "delete-label",
    name: "Delete label",
    method: "DELETE",
    path: "/api/v1/fleet/labels/:name",
  },
  {
    id: "list-users",
    name: "List users",
    method: "GET",
    path: "/api/v1/fleet/users",
  },
  {
    id: "create-user",
    name: "Create user",
    method: "POST",
    path: "/api/v1/fleet/users/admin",
  },
  {
    id: "get-user",
    name: "Get user",
    method: "GET",
    path: "/api/v1/fleet/users/:id",
  },
  {
    id: "modify-user",
    name: "Modify user",
    method: "PATCH",
    path: "/api/v1/fleet/users/:id",
  },
  {
    id: "delete-user",
    name: "Delete user",
    method: "DELETE",
    path: "/api/v1/fleet/users/:id",
  },
  {
    id: "list-teams",
    name: "List fleets",
    method: "GET",
    path: "/api/v1/fleet/teams",
  },
  {
    id: "create-team",
    name: "Create fleet",
    method: "POST",
    path: "/api/v1/fleet/teams",
  },
  {
    id: "delete-team",
    name: "Delete fleet",
    method: "DELETE",
    path: "/api/v1/fleet/teams/:id",
  },
  {
    id: "list-software-titles",
    name: "List software titles",
    method: "GET",
    path: "/api/v1/fleet/software/titles",
  },
  {
    id: "list-policies",
    name: "List policies",
    method: "GET",
    path: "/api/v1/fleet/policies",
  },
  {
    id: "create-policy",
    name: "Create policy",
    method: "POST",
    path: "/api/v1/fleet/policies",
  },
  {
    id: "delete-policy",
    name: "Delete policy",
    method: "POST",
    path: "/api/v1/fleet/policies/delete",
  },
  {
    id: "run-live-query",
    name: "Run live query",
    method: "POST",
    path: "/api/v1/fleet/queries/run",
    deprecated: true,
  },
  {
    id: "list-queries",
    name: "List reports",
    method: "GET",
    path: "/api/v1/fleet/queries",
  },
  {
    id: "get-query",
    name: "Get report",
    method: "GET",
    path: "/api/v1/fleet/queries/:id",
  },
  {
    id: "list-packs",
    name: "List packs",
    method: "GET",
    path: "/api/v1/fleet/packs",
    deprecated: true,
  },
  {
    id: "get-config",
    name: "Get configuration",
    method: "GET",
    path: "/api/v1/fleet/config",
  },
  {
    id: "modify-config",
    name: "Modify configuration",
    method: "PATCH",
    path: "/api/v1/fleet/config",
  },
  {
    id: "list-activities",
    name: "List activities",
    method: "GET",
    path: "/api/v1/fleet/activities",
  },
  {
    id: "get-host-software",
    name: "Get host's software",
    method: "GET",
    path: "/api/v1/fleet/hosts/:id/software",
  },
  {
    id: "list-vulnerabilities",
    name: "List vulnerabilities",
    method: "GET",
    path: "/api/v1/fleet/vulnerabilities",
  },
  {
    id: "get-vulnerability",
    name: "Get vulnerability",
    method: "GET",
    path: "/api/v1/fleet/vulnerabilities/:cve",
  },
];

export default FLEET_API_ENDPOINTS;
