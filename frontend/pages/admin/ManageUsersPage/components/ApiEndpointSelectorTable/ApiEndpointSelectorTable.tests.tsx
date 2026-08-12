import React from "react";

import { screen, waitFor, within } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import apiEndpointsAPI from "services/entities/api_endpoints";
import { IApiEndpoint } from "interfaces/api_endpoint";

import ApiEndpointSelectorTable from "./ApiEndpointSelectorTable";

jest.mock("services/entities/api_endpoints");

const LIST_HOSTS: IApiEndpoint = {
  method: "GET",
  path: "/api/v1/fleet/hosts",
  display_name: "List hosts",
  deprecated: false,
};

const UNINSTALL_SOFTWARE: IApiEndpoint = {
  method: "POST",
  path: "/api/v1/fleet/hosts/:id/software/:software_title_id/uninstall",
  display_name: "Uninstall software",
  deprecated: false,
};

const GET_HOST_SOFTWARE: IApiEndpoint = {
  method: "GET",
  path: "/api/v1/fleet/hosts/:id/software",
  display_name: "List host's software",
  deprecated: false,
};

const DEPRECATED_ENDPOINT: IApiEndpoint = {
  method: "GET",
  path: "/api/v1/fleet/packs",
  display_name: "List packs",
  deprecated: true,
};

const MOCK_ENDPOINTS: IApiEndpoint[] = [
  UNINSTALL_SOFTWARE,
  GET_HOST_SOFTWARE,
  DEPRECATED_ENDPOINT,
  LIST_HOSTS,
];

describe("ApiEndpointSelectorTable", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    (apiEndpointsAPI.loadAll as jest.Mock).mockResolvedValue(MOCK_ENDPOINTS);
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it("does not show a results dropdown when the search box is empty", async () => {
    render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={jest.fn()}
      />
    );

    await waitFor(() => expect(apiEndpointsAPI.loadAll).toHaveBeenCalled());
    expect(screen.queryByText("List hosts")).not.toBeInTheDocument();
  });

  it("ranks a broad, single-word search by relevance instead of catalog order", async () => {
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={jest.fn()}
      />
    );

    await user.type(
      screen.getByPlaceholderText("Search by name or path"),
      "hosts"
    );

    const names = await screen.findAllByText(
      /^(List hosts|List host's software|Uninstall software)$/
    );
    // "List hosts" is a whole-word match on a shallower path than the other
    // two "hosts"-containing endpoints, so it should rank first.
    expect(names[0]).toHaveTextContent("List hosts");
  });

  it("ranks an exact name match first even when it isn't first in catalog order", async () => {
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={jest.fn()}
      />
    );

    await user.type(
      screen.getByPlaceholderText("Search by name or path"),
      "list hosts"
    );

    const results = await screen.findAllByText(
      /^(List hosts|List host's software)$/
    );
    expect(results).toHaveLength(1);
    expect(results[0]).toHaveTextContent("List hosts");
  });

  it("matches on path as well as name", async () => {
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={jest.fn()}
      />
    );

    await user.type(
      screen.getByPlaceholderText("Search by name or path"),
      "uninstall"
    );

    await screen.findByText("Uninstall software");
    expect(screen.queryByText("List hosts")).not.toBeInTheDocument();
  });

  it("excludes already-selected endpoints from the search results", async () => {
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[
          { method: LIST_HOSTS.method, path: LIST_HOSTS.path },
        ]}
        onSelectionChange={jest.fn()}
      />
    );

    await user.type(
      screen.getByPlaceholderText("Search by name or path"),
      "hosts"
    );

    await screen.findByText("Uninstall software");
    // "List hosts" should only appear once now, in the selected-endpoints
    // table, not also in the search results dropdown.
    expect(screen.getAllByText("List hosts")).toHaveLength(1);
  });

  it("shows an empty state when nothing matches", async () => {
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={jest.fn()}
      />
    );

    await user.type(
      screen.getByPlaceholderText("Search by name or path"),
      "nonexistent-endpoint"
    );

    await screen.findByText("No matching API endpoints.");
  });

  it("shows a deprecated badge for deprecated endpoints", async () => {
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={jest.fn()}
      />
    );

    await user.type(
      screen.getByPlaceholderText("Search by name or path"),
      "packs"
    );

    await screen.findByText("List packs");
    expect(screen.getByText("Deprecated")).toBeInTheDocument();
  });

  it("adds the clicked endpoint to the selection and clears the search text", async () => {
    const onSelectionChange = jest.fn();
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[]}
        onSelectionChange={onSelectionChange}
      />
    );

    const searchInput = screen.getByPlaceholderText("Search by name or path");
    await user.type(searchInput, "list hosts");

    const result = await screen.findByText("List hosts");
    await user.click(result);

    await waitFor(() => {
      expect(onSelectionChange).toHaveBeenCalledWith([
        { method: LIST_HOSTS.method, path: LIST_HOSTS.path },
      ]);
    });
    expect(searchInput).toHaveValue("");
  });

  it("removes an endpoint from the selected-endpoints table", async () => {
    const onSelectionChange = jest.fn();
    const { user } = render(
      <ApiEndpointSelectorTable
        selectedEndpoints={[
          { method: LIST_HOSTS.method, path: LIST_HOSTS.path },
        ]}
        onSelectionChange={onSelectionChange}
      />
    );

    const selectedRow = (await screen.findByText("List hosts")).closest("tr");
    if (!selectedRow) {
      throw new Error("Expected to find the selected endpoint's table row");
    }

    await user.click(within(selectedRow).getByRole("button"));

    expect(onSelectionChange).toHaveBeenCalledWith([]);
  });
});
