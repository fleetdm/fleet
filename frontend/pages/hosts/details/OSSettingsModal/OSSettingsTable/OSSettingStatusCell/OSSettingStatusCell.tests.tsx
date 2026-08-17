import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  IHostMdmProfile,
  ProfileOperationType,
} from "interfaces/mdm";
import { HOST_NAME_SYNTHETIC_PROFILE_UUID } from "pages/hosts/details/helpers";
import OSSettingStatusCell from "./OSSettingStatusCell";
import { ANDROID_CERT_RETRYING_DISPLAY_CONFIG } from "./helpers";

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

  // Android certificate templates Fleet is automatically retrying after a failure. These keep an
  // in-progress status while carrying the detail from the failed attempt.
  describe("retrying Android certificate", () => {
    // What the Fleet Android app reports, and how the UI restates it.
    const SCEP_ERROR =
      "Network error during SCEP enrollment: Failed to communicate with SCEP server";
    const SCEP_MESSAGE = "Fleet couldn't reach the certificate authority.";

    const createRetryingCertProfile = (
      overrides?: Partial<IHostMdmProfile>
    ): IHostMdmProfile =>
      createMockHostMdmProfile({
        profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
        name: "BeyondCorp",
        platform: "android",
        operation_type: "install",
        status: "delivered",
        detail: SCEP_ERROR,
        certificate_template_id: 4,
        retrying: true,
        retry_count: 1,
        max_retries: 3,
        ...overrides,
      });

    const renderStatusCell = (profile: IHostMdmProfile) =>
      createCustomRenderer()(
        <OSSettingStatusCell
          profileName={profile.name}
          status={profile.status}
          operationType={profile.operation_type}
          hostPlatform="android"
          profileUUID={profile.profile_uuid}
          profile={profile}
        />
      );

    it.each(["pending", "delivering", "delivered"] as const)(
      "displays 'Retrying' with the error and attempt count for '%s' status",
      async (status) => {
        const profile = createRetryingCertProfile({ status });
        const { user } = renderStatusCell(profile);

        const statusText = screen.getByText("Retrying");
        expect(statusText).toBeInTheDocument();

        await user.hover(statusText);
        await waitFor(() => {
          // The attempt numbers count the initial attempt, so one retry of a maximum three
          // means this is attempt two of four.
          expect(
            screen.getByText(
              `${SCEP_MESSAGE} Retrying enrollment (attempt 2 of 4).`
            )
          ).toBeInTheDocument();
        });
      }
    );

    // Product decided against a new status icon for retries, since it would have to be
    // introduced across the rest of the OS settings UI. Retries reuse the in-progress icon and
    // are distinguished by the status text and tooltip alone.
    it("reuses the in-progress icon rather than introducing a new one", () => {
      expect(ANDROID_CERT_RETRYING_DISPLAY_CONFIG?.iconName).toEqual(
        "pending-outline"
      );

      const { container: retrying } = renderStatusCell(
        createRetryingCertProfile()
      );
      const { container: enforcing } = renderStatusCell(
        createRetryingCertProfile({
          retrying: false,
          detail: "",
          retry_count: 0,
        })
      );

      const retryingIcon = retrying.querySelector("svg");
      const enforcingIcon = enforcing.querySelector("svg");
      // Both must actually render an icon, otherwise the comparison below is vacuous.
      expect(retryingIcon).not.toBeNull();
      expect(enforcingIcon).not.toBeNull();
      expect(retryingIcon?.outerHTML).toEqual(enforcingIcon?.outerHTML);
    });

    it("counts the final attempt when the retries are used up", async () => {
      const profile = createRetryingCertProfile({ retry_count: 3 });
      const { user } = renderStatusCell(profile);

      await user.hover(screen.getByText("Retrying"));
      await waitFor(() => {
        expect(
          screen.getByText(
            `${SCEP_MESSAGE} Retrying enrollment (attempt 4 of 4).`
          )
        ).toBeInTheDocument();
      });
    });

    it("does not repeat the period when an unrecognized detail already ends in one", async () => {
      const profile = createRetryingCertProfile({
        detail: "Something went wrong.",
      });
      const { user } = renderStatusCell(profile);

      await user.hover(screen.getByText("Retrying"));
      await waitFor(() => {
        expect(
          screen.getByText(
            "Something went wrong. Retrying enrollment (attempt 2 of 4)."
          )
        ).toBeInTheDocument();
      });
    });

    it("displays 'Enforcing' on a first delivery that has not failed", () => {
      const profile = createRetryingCertProfile({
        retrying: false,
        detail: "",
        retry_count: 0,
      });
      renderStatusCell(profile);

      expect(screen.getByText("Enforcing")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });

    it("displays 'Enforcing' after a manual resend, which clears the detail", () => {
      // A resend sets the retry count to the maximum so the next failure is terminal. The
      // server reports retrying: false for it, which is the only thing separating it from the
      // final automatic retry below.
      const profile = createRetryingCertProfile({
        retrying: false,
        detail: "",
        retry_count: 3,
      });
      renderStatusCell(profile);

      expect(screen.getByText("Enforcing")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });

    // Reporting a failure detail is optional, so a retry can carry no error message at all. The
    // retry itself is still worth surfacing.
    it("displays 'Retrying' when the host reported a failure without a detail", async () => {
      const profile = createRetryingCertProfile({ detail: "" });
      const { user } = renderStatusCell(profile);

      const statusText = screen.getByText("Retrying");
      expect(statusText).toBeInTheDocument();

      await user.hover(statusText);
      await waitFor(() => {
        expect(
          screen.getByText("Retrying enrollment (attempt 2 of 4).")
        ).toBeInTheDocument();
      });
    });

    // The final automatic retry looks exactly like a manual resend from the client's side: both
    // sit at the maximum retry count with no detail. Only the server can separate them, which is
    // why `retrying` is server-decided rather than inferred here.
    it("displays 'Retrying' on a final retry with no detail, unlike a resend", async () => {
      const profile = createRetryingCertProfile({
        detail: "",
        retry_count: 3,
      });
      const { user } = renderStatusCell(profile);

      const statusText = screen.getByText("Retrying");
      expect(statusText).toBeInTheDocument();

      await user.hover(statusText);
      await waitFor(() => {
        expect(
          screen.getByText("Retrying enrollment (attempt 4 of 4).")
        ).toBeInTheDocument();
      });
    });

    // Every reason the Fleet Android app reports gets restated without SCEP, CSR, or key pair
    // jargon, since these tooltips also reach the end user's My Device page.
    it.each([
      [
        "SCEP enrollment failed: challenge rejected",
        "The certificate authority rejected the request.",
      ],
      [
        "Certificate validation failed: bad chain",
        "The host couldn't validate the certificate from the certificate authority.",
      ],
      [
        "Failed to generate key pair: keystore error",
        "The host couldn't generate a private key.",
      ],
      [
        "Failed to create CSR: bad subject",
        "The host couldn't create a certificate signing request.",
      ],
      [
        "Invalid configuration: missing subject name",
        "The certificate configuration isn't valid.",
      ],
      [
        "Certificate installation failed for alias 'BeyondCorp': installKeyPair returned false",
        "The host couldn't install the certificate.",
      ],
    ])("restates %j in plain language", async (detail, message) => {
      const { user } = renderStatusCell(createRetryingCertProfile({ detail }));

      await user.hover(screen.getByText("Retrying"));
      await waitFor(() => {
        expect(
          screen.getByText(`${message} Retrying enrollment (attempt 2 of 4).`)
        ).toBeInTheDocument();
      });
    });

    it("falls back to the reported text for an unrecognized failure", async () => {
      const detail = "Unexpected error during enrollment: null";
      const { user } = renderStatusCell(createRetryingCertProfile({ detail }));

      await user.hover(screen.getByText("Retrying"));
      await waitFor(() => {
        expect(
          screen.getByText(`${detail}. Retrying enrollment (attempt 2 of 4).`)
        ).toBeInTheDocument();
      });
    });

    it("treats a blank detail the same as a missing one", async () => {
      const profile = createRetryingCertProfile({ detail: "   " });
      const { user } = renderStatusCell(profile);

      await user.hover(screen.getByText("Retrying"));
      await waitFor(() => {
        // Not ". Retrying enrollment (attempt 2 of 4)."
        expect(
          screen.getByText("Retrying enrollment (attempt 2 of 4).")
        ).toBeInTheDocument();
      });
    });

    it("displays 'Removing enforcement' for a removal, which is never retried", () => {
      // The server omits the retry fields entirely on a removal rather than sending them as
      // false/zero, so the row arrives without any of them.
      const profile = createRetryingCertProfile({
        retrying: undefined,
        retry_count: undefined,
        max_retries: undefined,
        operation_type: "remove",
      });
      renderStatusCell(profile);

      expect(screen.getByText("Removing enforcement")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });

    // The same wording as while retrying, so the failure is not reworded the moment Fleet gives up.
    it("still displays 'Failed', with the same message, once the retries are exhausted", async () => {
      const profile = createRetryingCertProfile({
        retrying: false,
        status: "failed",
        retry_count: 3,
      });
      const { user } = renderStatusCell(profile);

      const statusText = screen.getByText("Failed");
      expect(statusText).toBeInTheDocument();

      await user.hover(statusText);
      await waitFor(() => {
        expect(screen.getByText(SCEP_MESSAGE)).toBeInTheDocument();
      });
    });

    // Defensive only: this server always sends max_retries alongside retry_count.
    it("falls back to no attempt count if the retry allowance is missing", async () => {
      const profile = createRetryingCertProfile({ max_retries: undefined });
      const { user } = renderStatusCell(profile);

      await user.hover(screen.getByText("Retrying"));
      await waitFor(() => {
        expect(
          screen.getByText(`${SCEP_MESSAGE} Retrying enrollment.`)
        ).toBeInTheDocument();
      });
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

  describe("verified profiles with a detail", () => {
    // A custom activation's predicate can exclude a host. Fleet delivered the
    // profile correctly so the status is Verified, but the settings were not
    // applied, and the generic tooltip claims the opposite.
    it("shows the detail instead of the generic verified text", async () => {
      const customRender = createCustomRenderer();

      const detail =
        'Fleet verified, but predicate ("ASSET-002" == "ASSET-001") evaluated to false and settings were not applied to this host.';
      const profile = createMockHostMdmProfile({
        name: "Passcode",
        platform: "darwin",
        operation_type: "install",
        status: "verified",
        detail,
      });

      const { user } = customRender(
        <OSSettingStatusCell
          profileName="Passcode"
          status="verified"
          operationType="install"
          hostPlatform="darwin"
          profile={profile}
        />
      );

      await user.hover(screen.getByText("Verified"));

      await waitFor(() => {
        expect(screen.getByText(detail)).toBeInTheDocument();
      });
      expect(
        screen.queryByText("The host applied the setting. Fleet verified.")
      ).not.toBeInTheDocument();
    });

    it("keeps the generic verified tooltip when there is no detail", async () => {
      const customRender = createCustomRenderer();

      const profile = createMockHostMdmProfile({
        name: "Passcode",
        platform: "darwin",
        operation_type: "install",
        status: "verified",
        detail: "",
      });

      const { user } = customRender(
        <OSSettingStatusCell
          profileName="Passcode"
          status="verified"
          operationType="install"
          hostPlatform="darwin"
          profile={profile}
        />
      );

      await user.hover(screen.getByText("Verified"));
      await waitFor(() => {
        expect(
          screen.getByText("The host applied the setting. Fleet verified.")
        ).toBeInTheDocument();
      });
    });
  });
});
