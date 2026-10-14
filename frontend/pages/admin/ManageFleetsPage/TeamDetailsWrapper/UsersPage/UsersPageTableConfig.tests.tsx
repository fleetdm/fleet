import { render, screen } from "@testing-library/react";
import React from "react";

import createMockTeam from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import { IApiEndpointRef } from "interfaces/api_endpoint";

import {
  generateColumnConfigs,
  generateDataSet,
  ITeamUsersTableData,
} from "./UsersPageTableConfig";

const TEAM_ID = 7;

const mockEndpoints = (count: number): IApiEndpointRef[] =>
  Array.from({ length: count }, (_unused, i) => ({
    method: "GET",
    path: `/api/v1/fleet/endpoint-${i}`,
  }));

const createTeamUser = (
  overrides?: Parameters<typeof createMockUser>[0]
): ITeamUsersTableData => {
  const user = createMockUser({
    global_role: null,
    teams: [createMockTeam({ id: TEAM_ID, role: "observer" })],
    ...overrides,
  });
  const [row] = generateDataSet(TEAM_ID, [user]);
  return row;
};

const getPermissionsColumn = () =>
  generateColumnConfigs(jest.fn(), null).find((c) => c.id === "permissions");

const renderPermissionsCell = (row: ITeamUsersTableData) => {
  const Cell = getPermissionsColumn()?.Cell as (props: {
    cell: { value: string };
    row: { original: ITeamUsersTableData };
  }) => JSX.Element;

  render(<Cell cell={{ value: row.role }} row={{ original: row }} />);
};

describe("UsersPageTableConfig - API endpoint restrictions", () => {
  it("names the role column 'Permissions'", () => {
    const column = getPermissionsColumn();
    expect(column?.title).toBe("Permissions");
    expect(column?.Header).toBe("Permissions");
  });

  it("counts the endpoints a user is restricted to", () => {
    const row = createTeamUser({
      api_only: true,
      api_endpoints: mockEndpoints(3),
    });
    expect(row.apiEndpointCount).toBe(3);
  });

  it("counts zero endpoints for a user with unrestricted API access", () => {
    const row = createTeamUser({ api_only: true });
    expect(row.apiEndpointCount).toBe(0);
  });

  it("shows the fleet role and a badge with the endpoint count in the Permissions cell", () => {
    const row = createTeamUser({
      api_only: true,
      api_endpoints: mockEndpoints(16),
    });

    renderPermissionsCell(row);

    expect(screen.getByText("Observer")).toBeInTheDocument();
    expect(screen.getByText("16 API endpoints")).toBeInTheDocument();
  });

  it("singularizes the badge when the user is restricted to one endpoint", () => {
    const row = createTeamUser({
      api_only: true,
      api_endpoints: mockEndpoints(1),
    });

    renderPermissionsCell(row);

    expect(screen.getByText("1 API endpoint")).toBeInTheDocument();
  });

  it("omits the badge for a user with unrestricted API access", () => {
    const row = createTeamUser({ api_only: true });

    renderPermissionsCell(row);

    expect(screen.getByText("Observer")).toBeInTheDocument();
    expect(screen.queryByText(/API endpoint/)).not.toBeInTheDocument();
  });
});
