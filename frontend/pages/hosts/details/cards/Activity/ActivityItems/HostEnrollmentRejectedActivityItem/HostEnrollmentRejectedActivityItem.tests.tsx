import { render, screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType } from "interfaces/activity";

import HostEnrollmentRejectedActivityItem from "./HostEnrollmentRejectedActivityItem";

const renderItem = (reason?: string) =>
  render(
    <HostEnrollmentRejectedActivityItem
      activity={createMockHostPastActivity({
        type: ActivityType.HostEnrollmentRejected,
        actor_full_name: "",
        actor_id: 0,
        details: { reason, enrollment_plane: "orbit" },
      })}
      tab="past"
    />
  );

describe("HostEnrollmentRejectedActivityItem", () => {
  const cases: Array<[string | undefined, RegExp]> = [
    [
      "one_time_secret_spent",
      /one-time enroll secret was already used\. Resend the Fleetd configuration profile/i,
    ],
    [
      "one_time_secret_identifier_mismatch",
      /this host's one-time enroll secret with a different serial number or hardware UUID/i,
    ],
    [
      "shared_secret_for_mdm_managed_host",
      /used a shared enroll secret\. Hosts enrolled in Fleet MDM must enroll with their one-time enroll secret/i,
    ],
    [undefined, /^An enrollment attempt for this host was rejected\.$/i],
    ["something_new", /^An enrollment attempt for this host was rejected\.$/i],
  ];

  it.each(cases)("renders copy for reason %j", (reason, expected) => {
    renderItem(reason);
    expect(screen.getByText(expected)).toBeVisible();
  });
});
