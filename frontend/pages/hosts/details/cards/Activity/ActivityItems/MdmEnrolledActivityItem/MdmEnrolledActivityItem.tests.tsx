import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import { ActivityType } from "interfaces/activity";
import { Platform } from "interfaces/platform";

import MdmEnrolledActivityItem from "./MdmEnrolledActivityItem";

const renderItem = (platform: Platform, actor: string) =>
  render(
    <MdmEnrolledActivityItem
      activity={createMockHostPastActivity({
        type: ActivityType.MdmEnrolled,
        actor_full_name: actor,
        actor_id: actor ? 1 : 0,
        details: { platform },
      })}
      tab="past"
    />
  );

describe("MdmEnrolledActivityItem", () => {
  const cases: Array<[Platform, string, RegExp]> = [
    ["ios", "Admin User", /told Fleet to enroll this host/i],
    ["android", "Admin User", /told Fleet to enroll this host/i],
    ["android", "", /This host enrolled to Fleet/i],
    [
      "darwin",
      "Admin User",
      /told Fleet to turn on mobile device management \(MDM\) for this host/i,
    ],
    [
      "darwin",
      "",
      /Mobile device management \(MDM\) was turned on for this host/i,
    ],
  ];

  it.each(cases)("renders %s copy (actor=%j)", (platform, actor, expected) => {
    renderItem(platform, actor);
    if (actor) expect(screen.getByText(actor)).toBeVisible();
    expect(screen.getByText(expected)).toBeVisible();
  });

  it("does not render the cancel or show-details icons", () => {
    renderItem("darwin", "Admin User");
    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
