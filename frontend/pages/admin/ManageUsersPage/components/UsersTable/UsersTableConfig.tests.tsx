import React from "react";
import { render, screen } from "@testing-library/react";

import createMockUser from "__mocks__/userMock";
import { IInvite } from "interfaces/invite";
import { IApiEndpointRef } from "interfaces/api_endpoint";

import {
  combineDataSets,
  generateTableHeaders,
  IUserTableData,
} from "./UsersTableConfig";

const daysAgo = (days: number): string =>
  new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString();

const mockEndpoints = (count: number): IApiEndpointRef[] =>
  Array.from({ length: count }, (_unused, i) => ({
    method: "GET",
    path: `/api/v1/fleet/endpoint-${i}`,
  }));

const renderPermissionsCell = (row: IUserTableData) => {
  const column = generateTableHeaders(jest.fn(), true).find(
    (c) => c.id === "permissions"
  );
  const Cell = column?.Cell as (props: {
    cell: { value: string };
    row: { original: IUserTableData };
  }) => JSX.Element;

  render(<Cell cell={{ value: row.role }} row={{ original: row }} />);
};

const createMockInvite = (overrides?: Partial<IInvite>): IInvite => ({
  created_at: daysAgo(1),
  updated_at: daysAgo(1),
  id: 1,
  invited_by: 99,
  email: "invitee@example.com",
  name: "Invited User",
  sso_enabled: false,
  global_role: "observer",
  teams: [],
  ...overrides,
});

describe("UsersTableConfig - combineDataSets", () => {
  it("maps the server-computed 'active' status to 'Active'", () => {
    const users = [createMockUser({ status: "active" })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Active");
  });

  it("maps the server-computed 'inactive' status to 'Inactive'", () => {
    const users = [createMockUser({ status: "inactive" })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Inactive");
  });

  it("maps the server-computed 'no_access' status to 'No access'", () => {
    const users = [createMockUser({ status: "no_access" })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("No access");
  });

  it("defaults to 'Active' when the server did not compute a status", () => {
    const users = [createMockUser({ status: undefined })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.status).toBe("Active");
  });

  it("returns 'Invite pending' for invites", () => {
    const invites = [createMockInvite()];
    const [row] = combineDataSets([], invites, 99);
    expect(row.status).toBe("Invite pending");
  });

  it("returns 'No access' for invites without a role", () => {
    const invites = [createMockInvite({ global_role: null, teams: [] })];
    const [row] = combineDataSets([], invites, 99);
    expect(row.status).toBe("No access");
  });
});

describe("UsersTableConfig - API endpoint restrictions", () => {
  it("counts the endpoints a user is restricted to", () => {
    const users = [
      createMockUser({ api_only: true, api_endpoints: mockEndpoints(3) }),
    ];
    const [row] = combineDataSets(users, [], 99);
    expect(row.apiEndpointCount).toBe(3);
  });

  it("counts zero endpoints for a user with unrestricted API access", () => {
    const users = [createMockUser({ api_only: true })];
    const [row] = combineDataSets(users, [], 99);
    expect(row.apiEndpointCount).toBe(0);
  });

  it("counts zero endpoints for invites", () => {
    const [row] = combineDataSets([], [createMockInvite()], 99);
    expect(row.apiEndpointCount).toBe(0);
  });

  it("names the role column 'Permissions'", () => {
    const column = generateTableHeaders(jest.fn(), true).find(
      (c) => c.id === "permissions"
    );
    expect(column?.title).toBe("Permissions");
    expect(column?.Header).toBe("Permissions");
  });

  it("shows a badge with the endpoint count in the Permissions cell", () => {
    const users = [
      createMockUser({ api_only: true, api_endpoints: mockEndpoints(16) }),
    ];
    const [row] = combineDataSets(users, [], 99);

    renderPermissionsCell(row);

    expect(screen.getByText("Admin")).toBeInTheDocument();
    expect(screen.getByText("16 API endpoints")).toBeInTheDocument();
  });

  it("singularizes the badge when the user is restricted to one endpoint", () => {
    const users = [
      createMockUser({ api_only: true, api_endpoints: mockEndpoints(1) }),
    ];
    const [row] = combineDataSets(users, [], 99);

    renderPermissionsCell(row);

    expect(screen.getByText("1 API endpoint")).toBeInTheDocument();
  });

  it("omits the badge for a user with unrestricted API access", () => {
    const users = [createMockUser({ api_only: true })];
    const [row] = combineDataSets(users, [], 99);

    renderPermissionsCell(row);

    expect(screen.queryByText(/API endpoint/)).not.toBeInTheDocument();
  });
});
