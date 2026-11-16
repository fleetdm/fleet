import { screen } from "@testing-library/react";
import React from "react";

import { createMockHostPastActivity } from "__mocks__/activityMock";
import { ActivityType, IActivityDetails } from "interfaces/activity";
import { createCustomRenderer } from "test/test-utils";

import StoppedReportingMicrosoftEntraDeviceIdActivityItem from "./StoppedReportingMicrosoftEntraDeviceIdActivityItem";

const renderItem = (details: IActivityDetails) => {
  const render = createCustomRenderer();
  render(
    <StoppedReportingMicrosoftEntraDeviceIdActivityItem
      activity={createMockHostPastActivity({
        type: ActivityType.StoppedReportingMicrosoftEntraDeviceId,
        actor_full_name: "",
        actor_id: 0,
        created_at: "2026-01-01T00:00:00Z",
        details,
      })}
      tab="past"
    />
  );
};

describe("StoppedReportingMicrosoftEntraDeviceIdActivityItem", () => {
  it("explains a device ID dropped for lack of a trusted certificate and what to do in Entra", () => {
    renderItem({
      entra_device_id: "11111111-1111-4111-8111-111111111111",
      reason: "no_trusted_certificate",
    });

    expect(screen.getByText("Fleet")).toBeInTheDocument();
    expect(
      screen.getByText("11111111-1111-4111-8111-111111111111")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/no longer has a trusted certificate for it/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/disable the device in Entra/i)
    ).toBeInTheDocument();
  });

  it("names the replacement device ID", () => {
    renderItem({
      entra_device_id: "11111111-1111-4111-8111-111111111111",
      reason: "replaced",
      new_entra_device_id: "22222222-2222-4222-8222-222222222222",
    });

    expect(
      screen.getByText("22222222-2222-4222-8222-222222222222")
    ).toBeInTheDocument();
    expect(screen.getByText(/now reports/i)).toBeInTheDocument();
  });
});
