import React from "react";
import { screen } from "@testing-library/react";
import {
  createCustomRenderer,
  createMockRouter,
  waitForLoadingToFinish,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { createGetConfigHandler } from "test/handlers/config-handlers";

import createMockConfig, { DEFAULT_LICENSE_MOCK } from "__mocks__/configMock";

import IntegrationsPage from "./IntegrationsPage";

// TODO(jacob) - get config endpoint mock working so these tests accurately test Integrations page,
// which now gets its config from the API instead of context
describe("Integrations Page", () => {
  // TODO: change this test to cover rendering all other sections displayed.
  // describe("MDM", () => {
  //   it("renders the MDM sidenav and content if MDM feature is enabled", async () => {
  //     mockServer.use(createGetConfigHandler());
  //     const mockRouter = createMockRouter();
  //     const mockConfig = createMockConfig();

  //     const render = createCustomRenderer({
  //       withBackendMock: true,
  //       context: {
  //         app: {
  //           isMacMdmEnabledAndConfigured: true,
  //           config: mockConfig,
  //         },
  //       },
  //     });

  //     // await setTimeout(() => true, 1000);

  //     const { container } = render(
  //       <IntegrationsPage router={mockRouter} params={{ section: "mdm" }} />
  //     );

  // sidenav label, sidenav tooltip, and card header
  //     await waitForLoadingToFinish(container);

  //     expect(
  //       screen.getAllByText("Mobile device management (MDM)")
  //     ).toHaveLength(3);
  //   });
  // });
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
    // it("renders the Conditional access sidenav for managed cloud Fleet instances", () => {
    //   const mockRouter = createMockRouter();
    //   const mockConfig = createMockConfig();

    //   const render = createCustomRenderer({
    //     withBackendMock: true,
    //     context: {
    //       app: {
    //         config: mockConfig,
    //       },
    //     },
    //   });

    //   render(<IntegrationsPage router={mockRouter} params={{}} />);

    //   expect(screen.queryByText("Conditional access")).toBeInTheDocument();
    // });
  });
});
