import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import { ActivityType } from "interfaces/activity";
import { Platform } from "interfaces/platform";

import MdmUnenrolledActivityItem from "./MdmUnenrolledActivityItem";

const renderItem = (platform: Platform, actor: string) =>
  render(
    <MdmUnenrolledActivityItem
      activity={createMockHostPastActivity({
        type: ActivityType.MdmUnenrolled,
        actor_full_name: actor,
        actor_id: actor ? 1 : 0,
        details: { platform },
      })}
      tab="past"
    />
  );

describe("MdmUnenrolledActivityItem", () => {
  const cases: Array<[Platform, string, RegExp]> = [
    ["ios", "Admin User", /told Fleet to unenroll this host/i],
    ["android", "Admin User", /told Fleet to unenroll this host/i],
    ["android", "", /This host is unenrolled from Fleet/i],
    [
      "darwin",
      "Admin User",
      /told Fleet to turn off mobile device management \(MDM\) for this host/i,
    ],
    [
      "darwin",
      "",
      /Mobile device management \(MDM\) was turned off for this host/i,
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
