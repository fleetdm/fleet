import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";

import Passwords from "./Passwords";

const renderCard = (currentTeamId: number, recoveryLockEnabled = false) => {
  const onMutation = jest.fn();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        config: {
          mdm: {
            enabled_and_configured: true,
            macos_recovery_lock_password_enabled: recoveryLockEnabled,
          },
          gitops: { gitops_mode_enabled: false },
        },
      },
    },
  });
  const result = render(
    <Passwords
      currentTeamId={currentTeamId}
      router={createMockRouter()}
      onMutation={onMutation}
    />
  );
  return { ...result, onMutation };
};

describe("Passwords card", () => {
  it("reads and saves Recovery Lock for unassigned hosts in the global config", async () => {
    let savedBody: unknown;
    mockServer.use(
      http.patch(baseUrl("/config"), async ({ request }) => {
        savedBody = await request.json();
        return HttpResponse.json({});
      })
    );
    const { user, onMutation } = renderCard(0, true);

    const checkbox = screen.getByRole("checkbox");
    await waitFor(() => expect(checkbox).toBeChecked());

    await user.click(checkbox);
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onMutation).toHaveBeenCalled());
    expect(savedBody).toEqual({
      mdm: { macos_recovery_lock_password_enabled: false },
    });
  });

  it("reads and saves Recovery Lock for a fleet in the fleet's config", async () => {
    let savedBody: unknown;
    mockServer.use(
      http.get(baseUrl("/fleets/1"), () =>
        HttpResponse.json({
          fleet: {
            id: 1,
            name: "Workstations",
            mdm: { macos_recovery_lock_password_enabled: true },
          },
        })
      ),
      http.patch(baseUrl("/fleets/1"), async ({ request }) => {
        savedBody = await request.json();
        return HttpResponse.json({});
      })
    );
    const { user, onMutation } = renderCard(1);

    await waitFor(() => expect(screen.getByRole("checkbox")).toBeChecked());

    await user.click(screen.getByRole("checkbox"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onMutation).toHaveBeenCalled());
    expect(savedBody).toEqual({
      mdm: { macos_recovery_lock_password_enabled: false },
    });
  });
});
