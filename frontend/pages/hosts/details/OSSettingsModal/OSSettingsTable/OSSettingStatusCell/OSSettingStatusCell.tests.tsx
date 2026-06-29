import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
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
    await waitFor(() => {
      expect(screen.getByText(/verifying/)).toBeInTheDocument();
    });
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

    const statusText = screen.getByText("Enforcing");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host is running the command/)
      ).toBeInTheDocument();
    });
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

    const statusText = screen.getByText("Enforcing");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host is running the command/)
      ).toBeInTheDocument();
    });
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

    const statusText = screen.getByText("Enforcing");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host is running the command/)
      ).toBeInTheDocument();
    });
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

    const statusText = screen.getByText("Removing enforcement");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host is running the command/)
      ).toBeInTheDocument();
    });
  });
  it("Shows the profile detail in the tooltip when a pending Android profile is waiting for a certificate", async () => {
    const customRender = createCustomRenderer();

    const detailMessage =
      'Waiting for certificate "WiFi-Cert" to be installed on the host before applying this profile.';

    const profile = createMockHostMdmProfile({
      profile_uuid: "gf6dc58e8-d4c7-4d4b-8fa1-47de2bcb162c",
      name: "01-wifi-eap-tls-WiFi-Cert.onc",
      platform: "android",
      operation_type: "install",
      status: "pending",
      detail: detailMessage,
    });

    const { user } = customRender(
      <OSSettingStatusCell
        profileName={profile.name}
        status="pending"
        operationType="install"
        hostPlatform="android"
        profileUUID={profile.profile_uuid}
        profile={profile}
      />
    );

    const statusText = screen.getByText("Enforcing");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(screen.getByText(detailMessage)).toBeInTheDocument();
    });
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

    const statusText = screen.getByText("Removing enforcement");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host is running the command/)
      ).toBeInTheDocument();
    });
  });
});
