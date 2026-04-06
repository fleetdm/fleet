import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import {
  createMockSoftwareTitlesResponse,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import { noop } from "lodash";

import SoftwareLibraryTable from "./SoftwareLibraryTable";

const mockRouter = createMockRouter();

describe("Software table", () => {
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
        isSoftwareEnabled={false} // Set to false
        data={createMockSoftwareTitlesResponse({
          counts_updated_at: null,
          software_titles: [],
        })}
        query=""
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        softwareFilter="allSoftware"
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
    expect(screen.queryByText("All software")).toBeNull();
    expect(screen.queryByText("Available for install")).toBeNull();
  });

  it("Renders the page-wide empty state when no software are present hiding 'Available for install' filter", () => {
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
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        softwareFilter="allSoftware"
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see software? Check back later.")
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(screen.queryByText("Search")).toBeNull();
    expect(screen.queryByText("Updated")).toBeNull();
    expect(screen.queryByText("All software")).toBeNull();
    expect(screen.queryByText("Available for install")).toBeNull();
  });

  it("Renders the empty search state and 'Available for install' filter when search query does not exist but filter is applied", () => {
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
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        softwareFilter="installableSoftware" // Dropdown applied
        currentPage={0}
        teamId={1}
        isLoading={false}
      />
    );

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Expecting to see installable software? Check back later."
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Available for install")).toBeInTheDocument();
  });

  it("does not render 'Available for install' filter when team id is undefined (Fleet Free/All teams)", () => {
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
        perPage={20}
        orderDirection="asc"
        orderKey="hosts_count"
        softwareFilter="allSoftware"
        currentPage={0}
        teamId={undefined} // Undefined for Fleet Free or Fleet Premium "All teams"
        isLoading={false}
      />
    );

    expect(screen.queryByText("All software")).toBeNull();
    expect(screen.queryByText("Available for install")).toBeNull();
  });
});
