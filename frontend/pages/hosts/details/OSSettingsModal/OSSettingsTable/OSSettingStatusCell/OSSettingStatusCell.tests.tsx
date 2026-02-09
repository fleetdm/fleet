import React from "react";
import { render, screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  ProfileOperationType,
} from "interfaces/mdm";
import OSSettingStatusCell from "./OSSettingStatusCell";

describe("OS setting status cell", () => {
  it("Correctly displays the status text of a profile", () => {
    const status = "verifying";
    const operationType: ProfileOperationType = "install";

    render(
      <OSSettingStatusCell
        profileName="Test Profile"
        status={status}
        operationType={operationType}
      />
    );

    expect(screen.getByText("Verifying")).toBeInTheDocument();
  });

  it("Correctly displays the tooltip text for a profile", async () => {
    const status = "verifying";
    const operationType: ProfileOperationType = "install";

    const customRender = createCustomRenderer();

    const { user } = customRender(
      <OSSettingStatusCell
        profileName="Test Profile"
        status={status}
        operationType={operationType}
      />
    );

    const statusText = screen.getByText("Verifying");

    await user.hover(statusText);

    expect(screen.getByText(/verifying/)).toBeInTheDocument();
  });

  // Android cert statuses
  it("Displays Pending UI for 'pending' status with optype 'install'", async () => {
    const customRender = createCustomRenderer();

    const { user } = customRender(
      <OSSettingStatusCell
        profileName="Test cert"
        status="pending"
        operationType="install"
        hostPlatform="android"
        profileUUID={FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID}
      />
    );

    const statusText = screen.getByText("Enforcing (pending)");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    expect(
      screen.getByText(/The host is running the command/)
    ).toBeInTheDocument();
  });
  it("Displays Pending UI for 'delivering' status with optype 'install'", async () => {
    const customRender = createCustomRenderer();

    const { user } = customRender(
      <OSSettingStatusCell
        profileName="Test cert"
        status="delivering"
        operationType="install"
        hostPlatform="android"
        profileUUID={FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID}
      />
    );

    const statusText = screen.getByText("Enforcing (pending)");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    expect(
      screen.getByText(/The host is running the command/)
    ).toBeInTheDocument();
  });
  it("Displays Pending UI for 'delivered' status with optype 'install'", async () => {
    const customRender = createCustomRenderer();

    const { user } = customRender(
      <OSSettingStatusCell
        profileName="Test cert"
        status="delivered"
        operationType="install"
        hostPlatform="android"
        profileUUID={FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID}
      />
    );

    const statusText = screen.getByText("Enforcing (pending)");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    expect(
      screen.getByText(/The host is running the command/)
    ).toBeInTheDocument();
  });
  it("Displays Pending UI for 'delivering' status with optype 'remove'", async () => {
    const customRender = createCustomRenderer();

    const { user } = customRender(
      <OSSettingStatusCell
        profileName="Test cert"
        status="delivering"
        operationType="remove"
        hostPlatform="android"
        profileUUID={FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID}
      />
    );

    const statusText = screen.getByText("Removing enforcement (pending)");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    expect(
      screen.getByText(/The host is running the command/)
    ).toBeInTheDocument();
  });
  it("Displays Pending UI for 'delivered' status with optype 'remove'", async () => {
    const customRender = createCustomRenderer();

    const { user } = customRender(
      <OSSettingStatusCell
        profileName="Test cert"
        status="delivered"
        operationType="remove"
        hostPlatform="android"
        profileUUID={FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID}
      />
    );

    const statusText = screen.getByText("Removing enforcement (pending)");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    expect(
      screen.getByText(/The host is running the command/)
    ).toBeInTheDocument();
  });
});
