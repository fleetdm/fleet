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

const renderInventory = (pathname: string) => {
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
      filters={{
        vulnerable: false,
        types: ["macos_app", "brave_extension", "cursor_extension"],
      }}
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

  it.each([
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
  ])(
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
});
