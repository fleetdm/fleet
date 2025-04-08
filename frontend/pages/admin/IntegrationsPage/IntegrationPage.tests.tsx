import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import IntegrationsPage from "./IntegrationsPage";

describe("Integrations Page", () => {
  // TODO: change this test to cover rendering all other sections displayed.
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

    expect(screen.getAllByText("Mobile device management (MDM)")).toHaveLength(
      2
    );
  });
});
