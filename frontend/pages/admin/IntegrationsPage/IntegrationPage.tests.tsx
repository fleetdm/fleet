import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import IntegrationsPage from "./IntegrationsPage";

// TODO: figure out how to mock the router properly.
const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

describe("Integrations Page", () => {
  it("renders the MDM sidenav and content if MDM feature is enabled", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isMacMdmEnabledAndConfigured: true },
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
