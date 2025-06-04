import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  getLabelHandler,
  getLabelHostsHandler,
} from "test/handlers/label-handlers";

import EditLabelPage from "./EditLabelPage";

// TODO: make this a utility for other tests.
const generateMockRouterProps = (overrides?: any) => {
  return {
    location: {},
    params: {},
    route: {},
    router: [],
    routeParams: {},
    ...overrides,
  };
};

describe("EditLabelPage", () => {
  it("renders a message for build in labels", async () => {
    mockServer.use(getLabelHandler({ label_type: "builtin" }));
    const render = createCustomRenderer({ withBackendMock: true });

    const routerProps = generateMockRouterProps({
      routeParams: { label_id: "1" },
    });
    render(<EditLabelPage {...routerProps} />);

    // waiting for the message to render
    const builtinMessage = await screen.findByText(
      "Built in labels cannot be edited"
    );

    expect(builtinMessage).toBeInTheDocument();
  });

  it("renders the DynamicLabelForm when the label is dynamic", async () => {
    mockServer.use(getLabelHandler({ label_membership_type: "dynamic" }));
    const render = createCustomRenderer({ withBackendMock: true });

    const routerProps = generateMockRouterProps({
      routeParams: { label_id: "1" },
    });
    render(<EditLabelPage {...routerProps} />);

    // waiting for the message to render
    const queryLabel = await screen.findByText("Query");
    const platformLabel = await screen.findByText("Platform");

    expect(queryLabel).toBeInTheDocument();
    expect(platformLabel).toBeInTheDocument();
    expect(screen.getByText(/Label queries are immutable/)).toBeInTheDocument();
    expect(
      screen.getByText(/Label platforms are immutable/)
    ).toBeInTheDocument();
  });

  it("renders the ManualLabelForm when the label is manual", async () => {
    mockServer.use(getLabelHandler({ label_membership_type: "manual" }));
    mockServer.use(
      getLabelHostsHandler([
        {
          hostname: "hosty numero uno",
          display_name: "Test host #1",
          team_id: 2,
          team_name: "Mobile",
          platform: "ios",
          os_version: "iOS 14.7.1",
          hardware_serial: "test-serial-1",
        },
        {
          hostname: "hosty numero dos",
          display_name: "Test host #2",
          team_id: 2,
          team_name: "Mobile",
          platform: "ios",
          os_version: "iOS 14.7.1",
          hardware_serial: "test-serial-2",
        },
      ])
    );
    const render = createCustomRenderer({ withBackendMock: true });

    const routerProps = generateMockRouterProps({
      routeParams: { label_id: "1" },
    });
    render(<EditLabelPage {...routerProps} />);

    // waiting for the message to render
    const selectHostsLabel = await screen.findByText("Select hosts");

    expect(selectHostsLabel).toBeInTheDocument();

    // expect host info to be on the page
    const host1 = await screen.findByText("Test host #1");
    const host2 = await screen.findByText("Test host #2");
    expect(host1).toBeInTheDocument();
    expect(host2).toBeInTheDocument();
  });
});
