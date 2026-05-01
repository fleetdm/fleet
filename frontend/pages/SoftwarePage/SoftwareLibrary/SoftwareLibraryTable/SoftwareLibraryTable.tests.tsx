import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import { createMockSoftwareTitlesResponse } from "__mocks__/softwareMock";

import SoftwareLibraryTable from "./SoftwareLibraryTable";

const mockRouter = createMockRouter();

describe("Software library table", () => {
  it("Renders the page-wide disabled state when software inventory is disabled", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareLibraryTable
        router={mockRouter}
        isSoftwareEnabled={false}
        data={createMockSoftwareTitlesResponse({
          counts_updated_at: null,
          software_titles: [],
        })}
        query=""
        perPage={50}
        orderDirection="asc"
        orderKey="hosts_count"
        selfServiceOnly={false}
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
  });

  it("Renders the page-wide empty state when no software are present, hiding search", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareLibraryTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockSoftwareTitlesResponse({
          count: 0,
          counts_updated_at: null,
          software_titles: [],
        })}
        query=""
        perPage={50}
        orderDirection="asc"
        orderKey="hosts_count"
        selfServiceOnly={false}
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("No software available")).toBeInTheDocument();
    expect(
      screen.getByText("Add software to your library to get started.")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add software" })
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(screen.queryByText("Search")).toBeNull();
    expect(screen.queryByText("Self-service only")).toBeNull();
  });

  it("Renders the empty search state and self-service toggle when self-service filter is applied", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    render(
      <SoftwareLibraryTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockSoftwareTitlesResponse({
          counts_updated_at: null,
          software_titles: [],
        })}
        query=""
        perPage={50}
        orderDirection="asc"
        orderKey="hosts_count"
        selfServiceOnly
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(screen.getByText("Self-service only")).toBeInTheDocument();
  });

  it("Renders the empty state without Add software button for observers", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: false,
          isGlobalMaintainer: false,
          isTeamAdmin: false,
          isTeamMaintainer: false,
          currentUser: createMockUser({ global_role: "observer" }),
        },
      },
    });

    render(
      <SoftwareLibraryTable
        router={mockRouter}
        isSoftwareEnabled
        data={createMockSoftwareTitlesResponse({
          count: 0,
          counts_updated_at: null,
          software_titles: [],
        })}
        query=""
        perPage={50}
        orderDirection="asc"
        orderKey="hosts_count"
        selfServiceOnly={false}
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("No software available")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Software added to this fleet's library will appear here."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add software" })
    ).not.toBeInTheDocument();
  });
});
