import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { http, HttpResponse } from "msw";
import React from "react";

import { createMockActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";
import {
  activityHandlerHasMoreActivities,
  activityHandlerHasPreviousActivities,
} from "test/handlers/activity-handlers";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";

import ActivityFeed from "./ActivityFeed";

describe("Activity Feed", () => {
  it("renders the correct number of activities", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render
    await screen.findByText("Test User");

    expect(screen.getByText("Test User")).toBeInTheDocument();
    expect(screen.getByText("Test User 2")).toBeInTheDocument();
    expect(screen.getByText("Test User 3")).toBeInTheDocument();
  });

  it("hides pagination when there are only one page of activities", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render
    await screen.findByText("Test User");

    expect(screen.queryByText(/previous/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/next/i)).not.toBeInTheDocument();
  });

  it("enables next pagination when there are more activities", async () => {
    mockServer.use(activityHandlerHasMoreActivities);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render and pagination to be present
    await screen.findByRole("button", { name: "Next" });

    expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
  });

  it("enables previous pagination when there are more previous activities", async () => {
    mockServer.use(activityHandlerHasPreviousActivities);

    const render = createCustomRenderer({
      withBackendMock: true,
    });

    const { user } = render(
      <ActivityFeed
        setShowActivityFeedTitle={noop}
        setRefetchActivities={noop}
        isPremiumTier
        router={createMockRouter()}
      />
    );

    // waiting for the activity data to render
    await screen.findAllByText("Test User");

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(screen.getByRole("button", { name: "Previous" })).toBeEnabled();
  });

  it.each([
    { platform: "windows" as const, profile: "Fleetd enroll secret" },
    { platform: "darwin" as const, profile: "Fleetd configuration" },
  ])(
    "names the $profile profile in the details of a $platform spent-secret rejection",
    async ({ platform, profile }) => {
      mockServer.use(
        http.get(baseUrl("/activities"), () =>
          HttpResponse.json({
            activities: [
              createMockActivity({
                type: ActivityType.HostEnrollmentRejected,
                fleet_initiated: true,
                details: {
                  host_display_name: "Anna's laptop",
                  reason: "one_time_secret_spent",
                  platform,
                },
              }),
            ],
            meta: { has_next_results: false, has_previous_results: false },
          })
        )
      );
      const render = createCustomRenderer({ withBackendMock: true });

      const { user } = render(
        <ActivityFeed
          setShowActivityFeedTitle={noop}
          setRefetchActivities={noop}
          isPremiumTier
          router={createMockRouter()}
        />
      );

      await user.click(
        await screen.findByRole("button", { name: "show info" })
      );

      expect(await screen.findByText("Enrollment details")).toBeInTheDocument();
      expect(screen.getByText(profile)).toBeInTheDocument();
    }
  );
});
