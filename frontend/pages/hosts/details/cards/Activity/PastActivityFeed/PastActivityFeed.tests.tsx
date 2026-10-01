import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";
import { IHostPastActivitiesResponse } from "services/entities/activities";
import { createCustomRenderer } from "test/test-utils";

import PastActivityFeed from "./PastActivityFeed";

const installActivity = createMockHostPastActivity({
  type: ActivityType.InstalledSoftware,
  actor_full_name: "Fleet",
  fleet_initiated: true,
  details: {
    software_title: "Firefox",
    software_package: "Firefox.pkg",
    host_display_name: "Test Host",
    source: "apps",
    status: "installed",
    install_uuid: "uuid-123",
  },
});

const buildResponse = (
  activities = [installActivity]
): IHostPastActivitiesResponse => ({
  activities,
  meta: { has_next_results: false, has_previous_results: false },
});

const renderWith = (isPremiumTier: boolean) =>
  createCustomRenderer({ context: { app: { isPremiumTier } } })(
    <PastActivityFeed
      activities={buildResponse()}
      onShowDetails={noop}
      onNextPage={noop}
      onPreviousPage={noop}
    />
  );

describe("PastActivityFeed", () => {
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
