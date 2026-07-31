import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
  createMockLocation,
} from "test/test-utils";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import createMockUser from "__mocks__/userMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import teamsAPI from "services/entities/teams";

import TeamSettings from "./TeamSettings";

const URL_ERROR = /valid webhook destination URL/i;
const REQUIRED_URL_ERROR = /Please enter a valid webhook destination URL/i;
const HOST_EXPIRY_ERROR = /Host expiry window must be a positive number/i;
const ENABLE_WEBHOOK = "Enable host status webhook";
const ENABLE_HOST_EXPIRY = "Enable host expiry";
const URL_PLACEHOLDER = "https://server.com/example";

const disabledWebhook = {
  enable_host_status_webhook: false,
  destination_url: "",
  host_percentage: 1,
  days_count: 1,
};

const renderTeamSettings = (hostStatusWebhook = disabledWebhook) => {
  mockServer.use(
    createGetConfigHandler(),
    // teamsAPI.load(1) -> GET /api/latest/fleet/fleets/1; component does select: (d) => d.team
    http.get(baseUrl("/fleets/:id"), () =>
      HttpResponse.json({
        team: {
          id: 1,
          name: "Team 1",
          host_expiry_settings: {
            host_expiry_enabled: false,
            host_expiry_window: 0,
          },
          webhook_settings: { host_status_webhook: hostStatusWebhook },
          features: {
            historical_data: { uptime: true, vulnerabilities: true },
          },
        },
      })
    )
  );

  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        isOnGlobalTeam: true,
        currentUser: createMockUser({ global_role: "admin" }),
        availableTeams: [createMockTeamSummary({ id: 1, name: "Team 1" })],
        setCurrentTeam: jest.fn(),
      },
    },
  });

  return render(
    <TeamSettings
      location={createMockLocation({
        pathname: "/settings/teams/settings",
        search: "?fleet_id=1",
        query: { fleet_id: "1" },
      })}
      router={createMockRouter()}
    />
  );
};

describe("TeamSettings - host status webhook validation", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("does not show a URL error when the webhook is enabled (#40410)", async () => {
    const { user } = renderTeamSettings();
    await screen.findByText("Webhook settings");

    await user.click(screen.getByText(ENABLE_WEBHOOK));

    expect(screen.getByPlaceholderText(URL_PLACEHOLDER)).toBeInTheDocument();
    expect(screen.queryByText(URL_ERROR)).not.toBeInTheDocument();
  });

  it("shows a URL error when the field is blurred while empty", async () => {
    const { user } = renderTeamSettings();
    await screen.findByText("Webhook settings");

    await user.click(screen.getByText(ENABLE_WEBHOOK));
    await user.click(screen.getByPlaceholderText(URL_PLACEHOLDER));
    await user.tab();

    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();
  });

  it("blocks submit and shows an error when the URL is empty", async () => {
    const updateSpy = jest
      .spyOn(teamsAPI, "update")
      .mockResolvedValue({} as never);
    const { user } = renderTeamSettings();
    await screen.findByText("Webhook settings");

    await user.click(screen.getByText(ENABLE_WEBHOOK));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it("clears the error and submits once a valid URL is entered", async () => {
    const updateSpy = jest
      .spyOn(teamsAPI, "update")
      .mockResolvedValue({} as never);
    const { user } = renderTeamSettings();
    await screen.findByText("Webhook settings");

    await user.click(screen.getByText(ENABLE_WEBHOOK));
    await user.type(
      screen.getByPlaceholderText(URL_PLACEHOLDER),
      "https://example.com/hook"
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(updateSpy).toHaveBeenCalled());
    expect(screen.queryByText(URL_ERROR)).not.toBeInTheDocument();
  });

  // Regression: suppressing the webhook URL error on change must not disable
  // host-expiry on-change validation.
  it("still validates the host expiry window on change", async () => {
    const { user } = renderTeamSettings();
    await screen.findByText("Webhook settings");

    await user.click(screen.getByText(ENABLE_HOST_EXPIRY));

    expect(await screen.findByText(HOST_EXPIRY_ERROR)).toBeInTheDocument();
  });
});
