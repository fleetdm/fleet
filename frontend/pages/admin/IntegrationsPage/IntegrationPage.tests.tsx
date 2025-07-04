import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import createMockConfig, { DEFAULT_LICENSE_MOCK } from "__mocks__/configMock";

import IntegrationsPage from "./IntegrationsPage";

describe("Integrations Page", () => {
  // TODO: change this test to cover rendering all other sections displayed.
  describe("MDM", () => {
    it("renders the MDM sidenav and content if MDM feature is enabled", () => {
      const mockRouter = createMockRouter();
      const mockConfig = createMockConfig();

      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isMacMdmEnabledAndConfigured: true,
            config: mockConfig,
          },
        },
      });

      render(
        <IntegrationsPage router={mockRouter} params={{ section: "mdm" }} />
      );

      // sidenave label, sidenav tooltip, and card header
      expect(
        screen.getAllByText("Mobile device management (MDM)")
      ).toHaveLength(3);
    });
  });
  describe("Conditional access", () => {
    it("Does not render the conditional access sidenav for self-hosted Fleet instances", () => {
      const mockRouter = createMockRouter();
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

      render(<IntegrationsPage router={mockRouter} params={{}} />);

      expect(screen.queryByText("Conditional access")).toBeNull();
    });

    it("renders the Conditional access sidenav for managed cloud Fleet instances", () => {
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

      // sidenave label, sidenav tooltip
      expect(screen.getAllByText("Conditional access")).toHaveLength(2);
    });
  });
});
