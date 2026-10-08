import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { ActivityType, IHostUpcomingActivity } from "interfaces/activity";
import { IHostUpcomingActivitiesResponse } from "services/entities/activities";
import { createCustomRenderer } from "test/test-utils";

import UpcomingActivityFeed from "./UpcomingActivityFeed";

const installActivity: IHostUpcomingActivity = {
  uuid: "upcoming-1",
  created_at: "2026-01-01T00:00:00Z",
  actor_full_name: "Fleet",
  actor_id: 0,
  actor_gravatar: "",
  actor_email: "",
  actor_api_only: false,
  fleet_initiated: true,
  type: ActivityType.InstalledSoftware,
  details: {
    software_title: "Firefox",
    software_package: "Firefox.pkg",
    host_display_name: "Test Host",
    source: "apps",
    status: "pending_install",
    install_uuid: "uuid-123",
  },
};

const buildResponse = (
  activities = [installActivity]
): IHostUpcomingActivitiesResponse => ({
  count: activities.length,
  activities,
  meta: { has_next_results: false, has_previous_results: false },
});

const renderWith = (isPremiumTier: boolean) =>
  createCustomRenderer({ context: { app: { isPremiumTier } } })(
    <UpcomingActivityFeed
      activities={buildResponse()}
      canCancelActivities
      onShowDetails={noop}
      onCancel={noop}
      onNextPage={noop}
      onPreviousPage={noop}
    />
  );

describe("UpcomingActivityFeed", () => {
  it("hides Show details on install activities for Fleet Free", () => {
    renderWith(false);

    expect(
      screen.queryByRole("button", { name: /show info/i })
    ).not.toBeInTheDocument();
  });

  it("shows Show details on install activities for Fleet Premium", () => {
    renderWith(true);

    expect(
      screen.getByRole("button", { name: /show info/i })
    ).toBeInTheDocument();
  });
});
