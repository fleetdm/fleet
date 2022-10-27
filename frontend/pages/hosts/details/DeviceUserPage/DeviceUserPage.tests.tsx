import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import DeviceUserPage from "./DeviceUserPage";

describe("Device User Page", () => {
  it("renders the software empty message if the device has no software", async () => {
    const render = createCustomRenderer({
      withUserEvents: true,
      withBackendMock: true,
    });

    expect(true).toBeTruthy();

    // TODO: fix return type from render
    const { user }: any = render(
      <DeviceUserPage params={{ device_auth_token: "testToken" }} />
    );

    // waiting for the device data to render
    await screen.findByText("About");

    await user.click(screen.getByRole("tab", { name: "Software" }));

    expect(
      screen.getByText("No installed software detected on this host.")
    ).toBeInTheDocument();
  });
});
