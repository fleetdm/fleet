import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import {
  getLabelHandler,
  getLabelHostsHandler,
} from "test/handlers/label-handlers";
import createMockConfig from "__mocks__/configMock";
import labelsAPI from "services/entities/labels";

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
    expect(
      screen.getByText(/Label queries and platforms are immutable/)
    ).toBeInTheDocument();
  });

  it("renders the ManualLabelForm when the label is manual", async () => {
    mockServer.use(getLabelHandler({ label_membership_type: "manual" }));
    mockServer.use(
      getLabelHostsHandler([
        {
          id: 1,
          hostname: "hosty numero uno",
          display_name: "Test host #1",
          team_id: 2,
          team_name: "Mobile",
          platform: "ios",
          os_version: "iOS 14.7.1",
          hardware_serial: "test-serial-1",
        },
        {
          id: 2,
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
    await screen.findByText("Test host #1");
    await screen.findByText("Test host #2");
  });

  describe("saving a manual label", () => {
    // createMockConfig supplies the fields MainContent reads (license, MDM); the AppContext
    // value replaces initialState wholesale rather than merging into it.
    const gitOpsContext = {
      app: {
        config: createMockConfig({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/example/fleet-gitops",
            exceptions: { labels: false, software: false, secrets: false },
          },
        }),
      },
    };

    const noGitOpsContext = {
      app: { config: createMockConfig() },
    };

    const renderManualLabelPage = (context?: Record<string, unknown>) => {
      mockServer.use(getLabelHandler({ label_membership_type: "manual" }));
      mockServer.use(
        getLabelHostsHandler([
          {
            id: 1,
            hostname: "hosty numero uno",
            display_name: "Test host #1",
            hardware_serial: "test-serial-1",
          },
        ])
      );
      const render = createCustomRenderer({
        withBackendMock: true,
        ...(context ? { context } : {}),
      });
      return render(
        <EditLabelPage
          {...generateMockRouterProps({ routeParams: { label_id: "1" } })}
        />
      );
    };

    beforeEach(() => {
      jest.spyOn(labelsAPI, "update").mockResolvedValue({ label: {} } as never);
    });

    afterEach(() => {
      jest.restoreAllMocks();
    });

    it("sends a membership-only update for a GitOps-managed manual label", async () => {
      const { user } = renderManualLabelPage(gitOpsContext);

      await screen.findByText("Select hosts");
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => {
        expect(labelsAPI.update).toHaveBeenCalledWith(
          1,
          expect.anything(),
          expect.objectContaining({ membershipOnly: true })
        );
      });
    });

    it("sends the full update when GitOps mode is off", async () => {
      const { user } = renderManualLabelPage(noGitOpsContext);

      await screen.findByText("Select hosts");
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => {
        expect(labelsAPI.update).toHaveBeenCalledWith(
          1,
          expect.anything(),
          expect.objectContaining({ membershipOnly: false })
        );
      });
    });
  });
});
