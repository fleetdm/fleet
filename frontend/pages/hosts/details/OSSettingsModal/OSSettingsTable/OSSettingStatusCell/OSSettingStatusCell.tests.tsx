import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  ProfileOperationType,
} from "interfaces/mdm";
import { HOST_NAME_SYNTHETIC_PROFILE_UUID } from "pages/hosts/details/helpers";
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

  it("Shows the profile detail in the tooltip when a verified Apple profile has a predicate detail", async () => {
    const customRender = createCustomRenderer();

    const detailMessage =
      'Fleet verified, but predicate (status.device.model.family == "AppleTV") evaluated to false and settings were not applied to this host.';

    const profile = createMockHostMdmProfile({
      profile_uuid: "d999270d7-f4bc-42d3-b1ae-6fe131d93b81",
      name: "Software Update settings",
      platform: "darwin",
      operation_type: "install",
      status: "verified",
      detail: detailMessage,
    });

    const { user } = customRender(
      <OSSettingStatusCell
        profileName={profile.name}
        status="verified"
        operationType="install"
        hostPlatform="darwin"
        profileUUID={profile.profile_uuid}
        profile={profile}
      />
    );

    const statusText = screen.getByText("Verified");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(screen.getByText(detailMessage)).toBeInTheDocument();
    });
  });

  it("Shows the standard tooltip when a verified Apple profile has no predicate detail", async () => {
    const customRender = createCustomRenderer();

    const profile = createMockHostMdmProfile({
      profile_uuid: "d999270d7-f4bc-42d3-b1ae-6fe131d93b81",
      platform: "darwin",
      operation_type: "install",
      status: "verified",
      detail: "This is verified",
    });

    const { user } = customRender(
      <OSSettingStatusCell
        profileName={profile.name}
        status="verified"
        operationType="install"
        hostPlatform="darwin"
        profileUUID={profile.profile_uuid}
        profile={profile}
      />
    );

    const statusText = screen.getByText("Verified");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host applied the setting. Fleet verified./)
      ).toBeInTheDocument();
    });
  });

  it("Shows the standard tooltip when a verified non-DDM Apple profile has a predicate detail", async () => {
    const customRender = createCustomRenderer();

    const profile = createMockHostMdmProfile({
      profile_uuid: "a999270d7-f4bc-42d3-b1ae-6fe131d93b81",
      platform: "darwin",
      operation_type: "install",
      status: "verified",
      detail:
        'Fleet verified, but predicate (status.device.model.family == "AppleTV") evaluated to false and settings were not applied to this host.',
    });

    const { user } = customRender(
      <OSSettingStatusCell
        profileName={profile.name}
        status="verified"
        operationType="install"
        hostPlatform="darwin"
        profileUUID={profile.profile_uuid}
        profile={profile}
      />
    );

    const statusText = screen.getByText("Verified");
    expect(statusText).toBeInTheDocument();

    await user.hover(statusText);
    await waitFor(() => {
      expect(
        screen.getByText(/The host applied the setting. Fleet verified./)
      ).toBeInTheDocument();
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

  // Host name template synthetic row
  describe("host name template row", () => {
    it("displays 'Enforcing' for pending status", async () => {
      const customRender = createCustomRenderer();

      const { user } = customRender(
        <OSSettingStatusCell
          profileName="Host name"
          status="pending"
          operationType={null}
          hostPlatform="darwin"
          profileUUID={HOST_NAME_SYNTHETIC_PROFILE_UUID}
        />
      );

      const statusText = screen.getByText("Enforcing");
      expect(statusText).toBeInTheDocument();

      await user.hover(statusText);
      await waitFor(() => {
        expect(
          screen.getByText(/Fleet is enforcing this fleet's host name template/)
        ).toBeInTheDocument();
      });
    });

    it("displays 'Verifying' for verifying status", () => {
      render(
        <OSSettingStatusCell
          profileName="Host name"
          status="verifying"
          operationType={null}
          hostPlatform="ios"
          profileUUID={HOST_NAME_SYNTHETIC_PROFILE_UUID}
        />
      );

      expect(screen.getByText("Verifying")).toBeInTheDocument();
    });

    it("displays 'Verified' for verified status", () => {
      render(
        <OSSettingStatusCell
          profileName="Host name"
          status="verified"
          operationType={null}
          hostPlatform="ipados"
          profileUUID={HOST_NAME_SYNTHETIC_PROFILE_UUID}
        />
      );

      expect(screen.getByText("Verified")).toBeInTheDocument();
    });

    it("displays 'Failed' and shows the profile detail in the tooltip", async () => {
      const customRender = createCustomRenderer();

      const detail =
        "Host was renamed on the device and no longer matches the fleet's naming template.";
      const profile = createMockHostMdmProfile({
        profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
        name: "Host name",
        platform: "darwin",
        operation_type: null,
        status: "failed",
        detail,
      });

      const { user } = customRender(
        <OSSettingStatusCell
          profileName="Host name"
          status="failed"
          operationType={null}
          hostPlatform="darwin"
          profileUUID={HOST_NAME_SYNTHETIC_PROFILE_UUID}
          profile={profile}
        />
      );

      const statusText = screen.getByText("Failed");
      expect(statusText).toBeInTheDocument();

      // With a null failed tooltip config, the cell falls through to the
      // detail-based error tooltip (generateErrorTooltip).
      await user.hover(statusText);
      await waitFor(() => {
        expect(screen.getByText(detail)).toBeInTheDocument();
      });
    });
  });
});
