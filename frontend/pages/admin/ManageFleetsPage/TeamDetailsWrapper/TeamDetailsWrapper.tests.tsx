import { screen, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import { IGitOpsExceptions } from "interfaces/config";
import mockServer from "test/mock-server";
import {
  createCustomRenderer,
  baseUrl,
  createMockRouter,
} from "test/test-utils";

import TeamDetailsWrapper from "./TeamDetailsWrapper";

// Avoid depending on react-router's browserHistory inside BackButton.
jest.mock("components/BackButton", () => ({
  __esModule: true,
  default: ({ text }: { text: string }) => (
    <button type="button">{text}</button>
  ),
}));

const TEAM = { id: 1, name: "Fleet 1", description: "", host_count: 0 };

const setupHandlers = () =>
  mockServer.use(
    http.get(baseUrl("/me"), () =>
      HttpResponse.json({ user: createMockUser({ global_role: "admin" }) })
    ),
    http.get(baseUrl("/fleets"), () => HttpResponse.json({ teams: [TEAM] })),
    http.get(baseUrl("/fleets/1/secrets"), () =>
      HttpResponse.json({ secrets: [] })
    )
  );

const renderPage = (
  gitOpsModeEnabled: boolean,
  exceptions?: IGitOpsExceptions
) => {
  setupHandlers();
  const config = createMockConfig();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
        isOnGlobalTeam: true,
        isPremiumTier: true,
        currentUser: createMockUser({ global_role: "admin" }),
        availableTeams: [createMockTeamSummary({ id: 1, name: "Fleet 1" })],
        setAvailableTeams: jest.fn(),
        setCurrentTeam: jest.fn(),
        setCurrentUser: jest.fn(),
        setUserSettings: jest.fn(),
        config: {
          ...config,
          gitops: {
            ...config.gitops,
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: "a.b.cc",
            // set unconditionally: the config mock defaults to excepting secrets
            exceptions,
          },
        },
      },
    },
  });

  return render(
    <TeamDetailsWrapper
      router={createMockRouter()}
      location={{
        pathname: "/settings/fleets/users",
        search: "?fleet_id=1",
        hash: "",
        query: { fleet_id: "1" },
      }}
    >
      <div />
    </TeamDetailsWrapper>
  );
};

// The actions also render as "More options" dropdown items for narrow viewports,
// so both surfaces are in the DOM and the buttons are scoped to explicitly.
const button = async (name: string) => {
  await screen.findByRole("button", { name: "Add hosts" });
  return within(
    document.querySelector(".action-buttons__secondary-buttons") as HTMLElement
  ).getByRole("button", { name });
};

describe("TeamDetailsWrapper", () => {
  it("keeps 'Manage enroll secrets' enabled in GitOps mode when secrets are excepted", async () => {
    renderPage(true, { labels: false, software: false, secrets: true });

    expect(await button("Manage enroll secrets")).toBeEnabled();
    expect(await button("Rename fleet")).toBeDisabled();
    expect(await button("Delete fleet")).toBeDisabled();
  });

  it("disables all three actions in GitOps mode when secrets are not excepted", async () => {
    renderPage(true, { labels: true, software: true, secrets: false });

    expect(await button("Manage enroll secrets")).toBeDisabled();
    expect(await button("Rename fleet")).toBeDisabled();
    expect(await button("Delete fleet")).toBeDisabled();
  });

  it("disables all three actions in GitOps mode when no exceptions are configured", async () => {
    renderPage(true);

    expect(await button("Manage enroll secrets")).toBeDisabled();
    expect(await button("Rename fleet")).toBeDisabled();
    expect(await button("Delete fleet")).toBeDisabled();
  });

  it("enables all three actions when GitOps mode is off", async () => {
    renderPage(false);

    expect(await button("Manage enroll secrets")).toBeEnabled();
    expect(await button("Rename fleet")).toBeEnabled();
    expect(await button("Delete fleet")).toBeEnabled();
  });
});
