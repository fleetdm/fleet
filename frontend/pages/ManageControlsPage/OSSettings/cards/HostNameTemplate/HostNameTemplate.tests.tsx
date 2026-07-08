import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { notify } from "components/ToastNotification";

import HostNameTemplate from "./HostNameTemplate";

const mockRouter = createMockRouter();

const teamHandler = (nameTemplate = "") =>
  http.get(baseUrl("/fleets/1"), () =>
    HttpResponse.json({
      team: { id: 1, name: "Team 1", mdm: { name_template: nameTemplate } },
      fleet: { id: 1, name: "Team 1", mdm: { name_template: nameTemplate } },
    })
  );

const baseProps = {
  currentTeamId: 1,
  router: mockRouter,
  onMutation: jest.fn(),
};

describe("HostNameTemplate card", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("renders the PremiumFeatureMessage on Free tier", () => {
    mockServer.use(teamHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: false,
          config: { mdm: { enabled_and_configured: true } },
        },
      },
    });

    render(<HostNameTemplate {...baseProps} />);

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
    expect(screen.queryByDisplayValue(/./)).not.toBeInTheDocument();
  });

  it("renders the MDM-not-configured empty state", () => {
    mockServer.use(teamHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: { mdm: { enabled_and_configured: false } },
        },
      },
    });

    render(<HostNameTemplate {...baseProps} />);

    expect(
      screen.getByText("MDM must be turned on to apply host name settings.")
    ).toBeInTheDocument();
  });

  it("loads and displays the current name template", async () => {
    mockServer.use(teamHandler("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: {
            mdm: { enabled_and_configured: true },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    render(<HostNameTemplate {...baseProps} />);

    await waitFor(() => {
      expect(
        screen.getByDisplayValue("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL")
      ).toBeInTheDocument();
    });
  });

  it("saves the template and fires onMutation", async () => {
    mockServer.use(teamHandler(""));
    let savedBody: unknown;
    mockServer.use(
      http.post(baseUrl("/host_name_template"), async ({ request }) => {
        savedBody = await request.json();
        return new HttpResponse(null, { status: 204 });
      })
    );
    const successSpy = jest.spyOn(notify, "success");

    const onMutation = jest.fn();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: {
            mdm: { enabled_and_configured: true },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    const { user } = render(
      <HostNameTemplate {...baseProps} onMutation={onMutation} />
    );

    // Save is disabled until the pristine form is edited.
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });

    await user.type(
      screen.getByRole("textbox"),
      "iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(onMutation).toHaveBeenCalled();
    });
    expect(savedBody).toEqual({
      fleet_id: 1,
      name_template: "iPad $FLEET_VAR_HOST_HARDWARE_SERIAL",
    });
    expect(successSpy).toHaveBeenCalledWith(
      "Successfully updated host name template."
    );
  });

  it("surfaces the server's 422 message verbatim on save error", async () => {
    mockServer.use(teamHandler(""));
    const serverMessage =
      "Fleet variable $FLEET_VAR_HOST_END_USER_IDP_GROUPS is not supported in host name templates.";
    mockServer.use(
      http.post(baseUrl("/host_name_template"), () =>
        HttpResponse.json(
          {
            message: "Validation Failed",
            errors: [{ name: "name_template", reason: serverMessage }],
          },
          { status: 422 }
        )
      )
    );
    const errorSpy = jest.spyOn(notify, "error");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: {
            mdm: { enabled_and_configured: true },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    const { user } = render(<HostNameTemplate {...baseProps} />);

    await waitFor(() => {
      expect(screen.getByRole("textbox")).toBeInTheDocument();
    });

    await user.type(
      screen.getByRole("textbox"),
      "$FLEET_VAR_HOST_END_USER_IDP_GROUPS"
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalledWith(serverMessage, expect.anything());
    });
  });

  it("keeps Save disabled while the form is pristine", async () => {
    mockServer.use(teamHandler("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: {
            mdm: { enabled_and_configured: true },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    const { user } = render(<HostNameTemplate {...baseProps} />);

    await waitFor(() => {
      expect(
        screen.getByDisplayValue("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL")
      ).toBeInTheDocument();
    });

    // Pristine: no changes yet.
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

    // Editing enables Save...
    await user.type(screen.getByRole("textbox"), " 2");
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("enables Save when a previously-set template is cleared", async () => {
    mockServer.use(teamHandler("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: {
            mdm: { enabled_and_configured: true },
            gitops: { gitops_mode_enabled: false },
          },
        },
      },
    });

    const { user } = render(<HostNameTemplate {...baseProps} />);

    await waitFor(() => {
      expect(
        screen.getByDisplayValue("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL")
      ).toBeInTheDocument();
    });

    await user.clear(screen.getByRole("textbox"));

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("disables the input in GitOps mode", async () => {
    mockServer.use(teamHandler("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          config: {
            mdm: { enabled_and_configured: true },
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "https://github.com/example/repo",
            },
          },
        },
      },
    });

    render(<HostNameTemplate {...baseProps} />);

    await waitFor(() => {
      expect(
        screen.getByDisplayValue("iPad $FLEET_VAR_HOST_HARDWARE_SERIAL")
      ).toBeDisabled();
    });
  });
});
