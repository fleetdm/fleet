import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import DeviceUserPage from "./DeviceUserPage";

describe("Device User Page", () => {
  it("renders the software empty message if the device has no software", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    // TODO: fix return type from render
    const { user } = render(
      <DeviceUserPage params={{ device_auth_token: "testToken" }} />
    );

    // waiting for the device data to render
    await screen.findByText("About");

    await user.click(screen.getByRole("tab", { name: "Software" }));

    // TODO: Fix this to the new copy
    // expect(screen.getByText("No software detected")).toBeInTheDocument();
  });
});
