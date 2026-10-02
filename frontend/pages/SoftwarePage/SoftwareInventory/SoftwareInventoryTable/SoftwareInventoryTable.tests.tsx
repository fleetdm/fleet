import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import {
  createMockSoftwareTitlesResponse,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import createMockUser from "__mocks__/userMock";
import PATHS from "router/paths";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareInventoryTable from "./SoftwareInventoryTable";

const renderTable = (
  props: Partial<React.ComponentProps<typeof SoftwareInventoryTable>> = {}
) => {
  const router = createMockRouter();
  const render = createCustomRenderer({
    // Rows render SoftwareIcon, which fetches through React Query.
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
        currentUser: createMockUser(),
      },
    },
  });
  const rendered = render(
    <SoftwareInventoryTable
      router={router}
      isSoftwareEnabled
      showVersions={false}
      data={createMockSoftwareTitlesResponse()}
      installableSoftwareExists={false}
      query=""
      perPage={20}
      orderDirection="asc"
      orderKey="hosts_count"
      filters={{ vulnerable: false }}
      currentPage={0}
      teamId={1}
      isLoading={false}
      onAddFiltersClick={noop}
      {...props}
    />
  );
  return { router, ...rendered };
};

const searchBox = () =>
  screen.getByPlaceholderText("Search by name or vulnerability (CVE)");

describe("Software inventory table", () => {
  it("Renders the page-wide disabled state when software inventory is disabled", () => {
    renderTable({
      isSoftwareEnabled: false,
      data: createMockSoftwareTitlesResponse({
        counts_updated_at: null,
        software_titles: [],
      }),
    });

    expect(screen.getByText("Software inventory disabled")).toBeInTheDocument();
    expect(screen.queryByText("Vulnerability")).toBeNull();
    expect(screen.queryByText("All software")).toBeNull();
    expect(screen.queryByText("Available for install")).toBeNull();
  });

  it("Renders the page-wide empty state when no software are present hiding search bar and vulnerability filtering", () => {
    renderTable({
      data: createMockSoftwareTitlesResponse({
        count: 0,
        counts_updated_at: null,
        software_titles: [],
      }),
    });

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Recently installed software will appear after the next scheduled check-in."
      )
    ).toBeInTheDocument();
    expect(screen.getByText("0 items")).toBeInTheDocument();
    expect(searchBox()).toBeDisabled();
    expect(screen.getByRole("button", { name: "Add filters" })).toBeDisabled();
    expect(screen.getByText("Show versions")).toBeInTheDocument();
  });

  it("Keeps controls enabled when versions toggle is applied but no data so users can toggle back", () => {
    renderTable({
      showVersions: true,
      data: createMockSoftwareVersionsResponse({
        counts_updated_at: null,
        software: [],
      }),
    });

    expect(screen.getByText("No software detected")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Recently installed software will appear after the next scheduled check-in."
      )
    ).toBeInTheDocument();
    // Controls stay enabled so users can toggle back to the titles view,
    // which may have installers even when the versions view is empty.
    expect(searchBox()).toBeEnabled();
    expect(screen.getByRole("button", { name: "Add filters" })).toBeEnabled();
  });

  it("Renders the empty search state and vulnerability filtering when search query does not exist but vulnerability filter is applied", () => {
    renderTable({
      data: createMockSoftwareTitlesResponse({
        counts_updated_at: null,
        software_titles: [],
      }),
      filters: { vulnerable: true },
    });

    expect(
      screen.getByText("No items match the current search criteria")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Expecting to see vulnerable software? Check back later."
      )
    ).toBeInTheDocument();
    expect(searchBox()).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Filtered" })
    ).toBeInTheDocument();
  });

  describe("type filter", () => {
    it("treats a type selection as filtered in the empty state", () => {
      renderTable({
        data: createMockSoftwareTitlesResponse({
          count: 0,
          software_titles: [],
        }),
        filters: { vulnerable: false, types: ["macos_app"] },
      });

      expect(
        screen.getByText("No items match the current search criteria")
      ).toBeInTheDocument();
      expect(searchBox()).toBeEnabled();
      expect(screen.getByRole("button", { name: "Filtered" })).toBeEnabled();
    });

    it("keeps types in the URL when the table query changes", async () => {
      const { router } = renderTable({
        filters: {
          vulnerable: false,
          types: ["macos_app", "brave_extension"],
        },
      });

      await waitFor(() => {
        expect(router.replace).toHaveBeenCalledWith(
          expect.stringContaining("types=brave_extension%2Cmacos_app")
        );
      });
    });

    it("keeps types when toggling Show versions", async () => {
      const { router, user } = renderTable({
        filters: { vulnerable: false, types: ["macos_app"] },
      });

      await user.click(screen.getByRole("switch"));

      expect(router.replace).toHaveBeenLastCalledWith(
        expect.stringMatching(
          new RegExp(`^${PATHS.SOFTWARE_VERSIONS}\\?.*types=macos_app`)
        )
      );
    });
  });
});
