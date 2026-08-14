import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import DeviceNotificationPage from "./DeviceNotificationPage";

describe("DeviceNotificationPage", () => {
  it("renders the route params (scaffold)", () => {
    const render = createCustomRenderer();

    render(
      <DeviceNotificationPage
        params={{
          device_auth_token: "test-token",
          notification_uuid: "test-uuid",
        }}
      />
    );

    expect(screen.getByText(/test-token/)).toBeInTheDocument();
    expect(screen.getByText(/test-uuid/)).toBeInTheDocument();
  });
});
