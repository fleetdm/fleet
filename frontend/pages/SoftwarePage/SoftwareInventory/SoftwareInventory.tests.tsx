import { waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import {
  createMockSoftwareTitlesResponse,
  createMockSoftwareVersionsResponse,
} from "__mocks__/softwareMock";
import createMockUser from "__mocks__/userMock";
import PATHS from "router/paths";
import softwareAPI from "services/entities/software";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareInventory from "./SoftwareInventory";
import { ISoftwareFilters } from "./SoftwareInventoryTable/helpers";

const renderInventory = (
  pathname: string,
  filters: ISoftwareFilters = {
    vulnerable: false,
    types: ["macos_app", "brave_extension", "cursor_extension"],
  }
) => {
  window.history.pushState({}, "", pathname);
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { isGlobalAdmin: true, currentUser: createMockUser() },
    },
  });
  render(
    <SoftwareInventory
      router={createMockRouter()}
      isSoftwareEnabled
      query=""
      perPage={20}
      orderDirection="desc"
      orderKey="hosts_count"
      filters={filters}
      currentPage={0}
      teamId={1}
      onAddFiltersClick={noop}
    />
  );
};

describe("SoftwareInventory", () => {
  afterEach(() => {
    jest.restoreAllMocks();
    window.history.pushState({}, "", "/");
  });

  const views = [
    {
      view: "titles",
      pathname: PATHS.SOFTWARE_INVENTORY,
      method: "getSoftwareTitles" as const,
      response: createMockSoftwareTitlesResponse(),
    },
    {
      view: "versions",
      pathname: PATHS.SOFTWARE_VERSIONS,
      method: "getSoftwareVersions" as const,
      response: createMockSoftwareVersionsResponse(),
    },
  ];

  it.each(views)(
    "sends the selected types to the $view endpoint as source and extension_for",
    async ({ pathname, method, response }) => {
      const spy = jest.spyOn(softwareAPI, method).mockResolvedValue(response);

      renderInventory(pathname);

      await waitFor(() => expect(spy).toHaveBeenCalled());
      const params = spy.mock.calls[0][0];
      expect(params).toEqual(
        expect.objectContaining({
          source: "apps,chrome_extensions,vscode_extensions",
          extension_for: "brave,cursor",
        })
      );
      expect(params).not.toHaveProperty("types");
    }
  );

  it.each(views)(
    "sends ai_tool to the $view endpoint with Vulnerable software off",
    async ({ pathname, method, response }) => {
      const spy = jest.spyOn(softwareAPI, method).mockResolvedValue(response);

      renderInventory(pathname, { vulnerable: false, aiTool: true });

      await waitFor(() => expect(spy).toHaveBeenCalled());
      expect(spy.mock.calls[0][0]).toEqual(
        expect.objectContaining({ aiTool: true, vulnerable: false })
      );
    }
  );

  it.each(views)(
    "omits ai_tool from the $view request when the filter is off",
    async ({ pathname, method, response }) => {
      const spy = jest.spyOn(softwareAPI, method).mockResolvedValue(response);

      renderInventory(pathname, { vulnerable: false, aiTool: false });

      await waitFor(() => expect(spy).toHaveBeenCalled());
      expect(spy.mock.calls[0][0].aiTool).toBeUndefined();
    }
  );

  // That request only picks the empty-state copy shown when no filter applies.
  it("omits ai_tool from the available-for-install check on an empty versions view", async () => {
    jest
      .spyOn(softwareAPI, "getSoftwareVersions")
      .mockResolvedValue(
        createMockSoftwareVersionsResponse({ count: 0, software: [] })
      );
    const titlesSpy = jest
      .spyOn(softwareAPI, "getSoftwareTitles")
      .mockResolvedValue(createMockSoftwareTitlesResponse());

    renderInventory(PATHS.SOFTWARE_VERSIONS, {
      vulnerable: false,
      aiTool: true,
    });

    await waitFor(() => expect(titlesSpy).toHaveBeenCalled());
    const params = titlesSpy.mock.calls[0][0];
    expect(params).toEqual(
      expect.objectContaining({ availableForInstall: true })
    );
    expect(params.aiTool).toBeUndefined();
  });
});
