import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
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

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const TEAM = { id: 1, name: "Fleet 1", description: "", host_count: 0 };

const setupHandlers = () =>
  mockServer.use(
    http.get(baseUrl("/me"), () =>
      HttpResponse.json({ user: createMockUser({ global_role: "admin" }) })
    ),
    http.get(baseUrl("/fleets"), () => HttpResponse.json({ fleets: [TEAM] })),
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

  describe("renaming the fleet", () => {
    const invalidNameResponse = (reason: string) =>
      HttpResponse.json(
        { message: "Validation Failed", errors: [{ name: "name", reason }] },
        { status: 422 }
      );

    const submitRename = async (name: string) => {
      const { user } = renderPage(false);
      await user.click(await button("Rename fleet"));
      const nameInput = screen.getByLabelText("Fleet name");
      await user.clear(nameInput);
      await user.type(nameInput, name);
      await user.click(screen.getByRole("button", { name: "Save" }));
    };

    beforeEach(() => {
      jest.clearAllMocks();
    });

    it("shows the conflict error under the name field on a 409, with no toast", async () => {
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), () =>
          HttpResponse.json(
            {
              message: "Conflict",
              errors: [
                {
                  name: "base",
                  reason:
                    'A fleet named "Workstations" already exists. Choose a fleet name distinct from other fleets.',
                },
              ],
            },
            { status: 409 }
          )
        )
      );

      await submitRename("work-stations");

      expect(
        await screen.findByText(
          "Choose a fleet name distinct from other fleets."
        )
      ).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("shows the letter-or-number error under the name field", async () => {
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), () =>
          invalidNameResponse("must include at least one letter or number")
        )
      );

      await submitRename("---");

      expect(
        await screen.findByText(
          "Fleet name must include at least one letter or number."
        )
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it.each(["Unassigned", "All fleets"])(
      "shows the reserved-name error under the name field for %s",
      async (name) => {
        mockServer.use(
          http.patch(baseUrl("/fleets/1"), () =>
            invalidNameResponse(`"${name}" is a reserved fleet name`)
          )
        );

        await submitRename(name);

        expect(
          await screen.findByText(
            `"${name}" is a reserved fleet name. Please try another name.`
          )
        ).toBeInTheDocument();
        expect(notify.error).not.toHaveBeenCalled();
      }
    );

    it("sends a rename to a variant of the fleet's own name", async () => {
      const patchSpy = jest.fn();
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), async ({ request }) => {
          patchSpy(await request.json());
          return HttpResponse.json({ fleet: { ...TEAM, name: "FLEET-1" } });
        })
      );

      await submitRename("FLEET-1");

      await waitFor(() => {
        expect(notify.success).toHaveBeenCalledWith(
          "Successfully updated fleet name to FLEET-1"
        );
      });
      expect(patchSpy).toHaveBeenCalledWith({ name: "FLEET-1" });
    });

    it("names the fleet in the toast for other errors", async () => {
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), () =>
          HttpResponse.json(
            { message: "error", errors: [{ name: "base", reason: "boom" }] },
            { status: 500 }
          )
        )
      );

      await submitRename("Servers");

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(
          "Could not rename Fleet 1. Please try again.",
          expect.anything()
        );
      });
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });
  });
});
