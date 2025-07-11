import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { createGetConfigHandler } from "test/handlers/config-handlers";

import createMockConfig, { DEFAULT_LICENSE_MOCK } from "__mocks__/configMock";

import IntegrationsPage from "./IntegrationsPage";

describe("Integrations Page", () => {
  // TODO: change this test to cover rendering all other sections displayed.
  describe("MDM", () => {
    it("renders the MDM sidenav and content if MDM feature is enabled", async () => {
      mockServer.use(createGetConfigHandler());

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            config: createMockConfig(),
          },
        },
      });

      render(
        <IntegrationsPage
          router={createMockRouter()}
          params={{ section: "mdm" }}
        />
      );

      expect(
        await screen.findAllByText("Mobile device management (MDM)")
      ).toHaveLength(3); // truncated side nav label, side nav label tooltip, card header
    });
  });
  describe("Conditional access", () => {
    it("Does not render the conditional access sidenav for self-hosted Fleet instances", () => {
      const mockConfig = createMockConfig({
        license: { ...DEFAULT_LICENSE_MOCK, managed_cloud: false },
      });

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: mockConfig,
          },
        },
      });

      render(<IntegrationsPage router={createMockRouter()} params={{}} />);

      expect(screen.queryByText("Conditional access")).toBeNull();
    });
    it("renders the Conditional access sidenav for managed cloud Fleet instances", async () => {
      mockServer.use(createGetConfigHandler());
      const mockRouter = createMockRouter();
      const mockConfig = createMockConfig();

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: mockConfig,
          },
        },
      });

      render(<IntegrationsPage router={mockRouter} params={{}} />);

      expect(await screen.findAllByText("Conditional access")).toHaveLength(2); // side nav label and card header
    });
  });
});
