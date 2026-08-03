import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createMockFleetMaintainedAppDetails } from "__mocks__/softwareMock";
import softwareAPI from "services/entities/software";
import teamPoliciesAPI from "services/entities/team_policies";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import FleetMaintainedAppDetailsPage from "./FleetMaintainedAppDetailsPage";

describe("FleetMaintainedAppDetailsPage", () => {
  beforeEach(() => {
    jest.spyOn(softwareAPI, "getFleetMaintainedApp").mockResolvedValue({
      fleet_maintained_app: createMockFleetMaintainedAppDetails(),
    });
    jest
      .spyOn(softwareAPI, "addFleetMaintainedApp")
      .mockResolvedValue({ software_title_id: 99 });
    jest.spyOn(teamPoliciesAPI, "create").mockResolvedValue({} as never);
  });

  afterEach(() => jest.restoreAllMocks());

  it("adds the FMA before creating its patch policy", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true } },
    });
    const router = createMockRouter();
    const { user } = render(
      <FleetMaintainedAppDetailsPage
        location={
          ({ query: { fleet_id: "3" } } as unknown) as React.ComponentProps<
            typeof FleetMaintainedAppDetailsPage
          >["location"]
        }
        router={router}
        routeParams={{ id: "1" }}
      />
    );

    await user.click(await screen.findByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("button", { name: "Add software" }));

    await waitFor(() => {
      expect(softwareAPI.addFleetMaintainedApp).toHaveBeenCalledWith(
        3,
        expect.objectContaining({ patch: true, patchOption: "closed" })
      );
      expect(teamPoliciesAPI.create).toHaveBeenCalledWith({
        team_id: 3,
        type: "patch",
        patch_software_title_id: 99,
        software_title_id: 99,
        patch_when_closed: true,
        continuous_automations_enabled: true,
      });
    });

    expect(
      (softwareAPI.addFleetMaintainedApp as jest.Mock).mock
        .invocationCallOrder[0]
    ).toBeLessThan(
      (teamPoliciesAPI.create as jest.Mock).mock.invocationCallOrder[0]
    );
  });

  it("navigates to the added software when patch policy creation fails", async () => {
    (teamPoliciesAPI.create as jest.Mock).mockRejectedValueOnce(
      new Error("Patch failed")
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true } },
    });
    const router = createMockRouter();
    const { user } = render(
      <FleetMaintainedAppDetailsPage
        location={
          ({ query: { fleet_id: "3" } } as unknown) as React.ComponentProps<
            typeof FleetMaintainedAppDetailsPage
          >["location"]
        }
        router={router}
        routeParams={{ id: "1" }}
      />
    );

    await user.click(await screen.findByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("button", { name: "Add software" }));

    await waitFor(() => {
      expect(softwareAPI.addFleetMaintainedApp).toHaveBeenCalledTimes(1);
      expect(teamPoliciesAPI.create).toHaveBeenCalledTimes(1);
      expect(router.push).toHaveBeenCalledWith(
        expect.stringMatching(/\/software\/titles\/99.*fleet_id=3/)
      );
    });
  });
});
