import React from "react";
import { screen } from "@testing-library/react";

import {
  createCustomRenderer,
  createMockLocation,
  createMockRouter,
} from "test/test-utils";
import IntegrationsPage from "./IntegrationsPage";

describe("Integrations Page", () => {
  // TODO: change this test to cover rendering all other sections displayed.
  it("renders the MDM sidenav and content if MDM feature is enabled", () => {
    const mockRouter = createMockRouter();
    const mockLocation = createMockLocation();

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isMacMdmEnabledAndConfigured: true },
      },
    });

    render(
      <IntegrationsPage
        router={mockRouter}
        params={{ section: "mdm" }}
        location={mockLocation}
      />
    );

    expect(screen.getAllByText("Mobile device management (MDM)")).toHaveLength(
      2
    );
  });
});
