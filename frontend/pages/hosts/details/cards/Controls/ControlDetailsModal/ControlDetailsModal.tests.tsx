import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockHostMdmProfile } from "__mocks__/hostMock";

import { FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID } from "interfaces/mdm";
import {
  generateRecoveryLockPasswordSetting,
  generateWinDiskEncryptionSetting,
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

import ControlDetailsModal from "./ControlDetailsModal";

const HOST = "Anna's MacBook Pro";

const renderModal = (
  props: Partial<React.ComponentProps<typeof ControlDetailsModal>> = {}
) => {
  const render = createCustomRenderer({ withBackendMock: true });

  return render(
    <ControlDetailsModal
      control={createMockHostMdmProfile({
        name: "Okta Verify settings",
        platform: "darwin",
        operation_type: "install",
        status: "verified",
        detail: "",
      })}
      hostDisplayName={HOST}
      canResendProfiles
      resendRequest={jest.fn()}
      onProfileResent={jest.fn()}
      onExit={jest.fn()}
      {...props}
    />
  );
};

describe("ControlDetailsModal", () => {
  it("names the host and the setting in the verified message", () => {
    renderModal();

    expect(screen.getByText(/applied/, { selector: "span" })).toHaveTextContent(
      "Anna's MacBook Pro applied Okta Verify settings."
    );
    expect(screen.getByText(/Fleet verified\./)).toBeInTheDocument();
  });

  it("drops the key phrasing for the FileVault profile when the fleet is enforce-only", () => {
    renderModal({
      isMacOSDiskEncryptionEnforceOnly: true,
      control: createMockHostMdmProfile({
        name: "Disk encryption",
        platform: "darwin",
        operation_type: "install",
        status: "verified",
        detail: "",
      }),
    });

    expect(
      screen.getByText(/turned disk encryption on\. Fleet verified\./)
    ).toBeInTheDocument();
    expect(screen.queryByText(/sent the key to Fleet/)).toBeNull();
  });

  it("drops the key retrieval phrasing while verifying when the fleet is enforce-only", () => {
    renderModal({
      isMacOSDiskEncryptionEnforceOnly: true,
      control: createMockHostMdmProfile({
        name: "Disk encryption",
        platform: "darwin",
        operation_type: "install",
        status: "verifying",
        detail: "",
      }),
    });

    expect(
      screen.getByText(
        /Fleet is verifying with osquery\. This may take up to one hour\./
      )
    ).toBeInTheDocument();
    expect(screen.queryByText(/retrieving the disk encryption key/)).toBeNull();
  });

  it("uses the disk encryption wording for the FileVault profile", () => {
    renderModal({
      control: createMockHostMdmProfile({
        name: "Disk encryption",
        platform: "darwin",
        operation_type: "install",
        status: "verified",
        detail: "",
      }),
    });

    expect(
      screen.getByText(
        /turned disk encryption on and sent the key to Fleet\. Fleet verified\./
      )
    ).toBeInTheDocument();
  });

  it("addresses the end user directly for action required on My device", () => {
    renderModal({
      isDeviceUser: true,
      control: createMockHostMdmProfile({
        name: "Okta Verify settings",
        platform: "darwin",
        operation_type: "install",
        status: "action_required",
        detail: "",
      }),
    });

    expect(screen.getByText(/Follow the/)).toBeInTheDocument();
    expect(screen.queryByText(/Ask the end user/)).not.toBeInTheDocument();
  });

  it("asks the admin to contact the end user for action required on host details", () => {
    renderModal({
      control: createMockHostMdmProfile({
        name: "Okta Verify settings",
        platform: "darwin",
        operation_type: "install",
        status: "action_required",
        detail: "",
      }),
    });

    expect(
      screen.getByText(/Ask the end user to follow the/)
    ).toBeInTheDocument();
  });

  describe("failed controls", () => {
    const failedWindowsProfile = createMockHostMdmProfile({
      name: "Edge policy",
      platform: "windows",
      operation_type: "install",
      status: "failed",
      detail:
        "./Device/Vendor/MSFT/Policy/Config/Fleet/A: status 200, ./Device/Vendor/MSFT/Policy/Config/Fleet/B: status 500",
    });

    it("shows the failure message and a copyable details block, one result per line", () => {
      renderModal({ control: failedWindowsProfile });

      expect(
        screen.getByText(/failed to apply/, { selector: "span" })
      ).toHaveTextContent("Anna's MacBook Pro failed to apply Edge policy.");
      expect(screen.getByText("Details:")).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Copy details" })
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          /Fleet\/A: status 200\s+\.\/Device\/Vendor\/MSFT\/Policy\/Config\/Fleet\/B: status 500/
        )
      ).toBeInTheDocument();
    });

    // A rewrite that drops the underlying error text (status codes, profile
    // IDs) still needs the raw output alongside it.
    it("renders actionable guidance above the raw detail when the error is recognized", () => {
      renderModal({
        control: createMockHostMdmProfile({
          name: "Wi-Fi",
          platform: "darwin",
          operation_type: "install",
          status: "failed",
          detail:
            "Couldn't get certificate from DigiCert. The API token configured in DIGICERT_TEST certificate authority is invalid.",
        }),
      });

      expect(screen.getByText(/correct it and resend/)).toBeInTheDocument();
      expect(screen.getByText("Details:")).toBeInTheDocument();
    });

    // Per the "Failed, no details" frame: guidance carrying a Learn more link
    // quotes the detail, so repeating it in the block below is noise.
    it("drops the details block when the guidance already quotes the detail", () => {
      renderModal({
        control: createMockHostMdmProfile({
          name: "android-bluetooth-disabled",
          platform: "android",
          operation_type: "install",
          status: "failed",
          detail:
            '"bluetoothDisabled" setting couldn\'t apply to a host. Reason: MANAGEMENT_MODE. Other settings are applied.',
        }),
      });

      expect(
        screen.getByText(/setting couldn.t apply to a host/)
      ).toBeInTheDocument();
      expect(screen.getByText(/Learn more/)).toBeInTheDocument();
      expect(screen.queryByText("Details:")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Copy details" })
      ).not.toBeInTheDocument();
    });

    it("drops the details block for an IdP email error", () => {
      renderModal({
        control: createMockHostMdmProfile({
          name: "Wi-Fi",
          platform: "darwin",
          operation_type: "install",
          status: "failed",
          detail: "There is no IdP email for this host.",
        }),
      });

      expect(screen.getByText(/Learn more/)).toBeInTheDocument();
      expect(screen.queryByText("Details:")).not.toBeInTheDocument();
    });

    it("shows a Resend action for a failed profile", () => {
      renderModal({ control: failedWindowsProfile });

      expect(
        screen.getByRole("button", { name: /Resend/ })
      ).toBeInTheDocument();
    });

    it("hides the Resend action when the page doesn't allow resending", () => {
      renderModal({
        control: failedWindowsProfile,
        canResendProfiles: false,
      });

      expect(
        screen.queryByRole("button", { name: /Resend/ })
      ).not.toBeInTheDocument();
    });

    it("closes the modal after a successful resend", async () => {
      const onProfileResent = jest.fn();
      const render = createCustomRenderer({ withBackendMock: true });

      const { user } = render(
        <ControlDetailsModal
          control={failedWindowsProfile}
          hostDisplayName={HOST}
          canResendProfiles
          resendRequest={jest.fn().mockResolvedValue(undefined)}
          onProfileResent={onProfileResent}
          onExit={jest.fn()}
        />
      );

      await user.click(screen.getByRole("button", { name: /Resend/ }));

      await waitFor(() => {
        expect(onProfileResent).toHaveBeenCalled();
      });
    });

    it("shows the appended retry note for a failed windows disk encryption row", () => {
      renderModal({
        control: generateWinDiskEncryptionSetting(
          "failed",
          "starting encryption: encrypt(C:): error code returned during encryption: -2147024809"
        ),
      });

      expect(
        screen.getByText(/failed to turn on disk encryption\./)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/Fleet will retry automatically\./)
      ).toBeInTheDocument();
    });
  });

  // These row types weren't mocked up in the Figma; product asked for them to
  // match the templated voice of the profile and disk-encryption copy.
  describe("synthesized row types", () => {
    it("names the host in the host name template message", () => {
      renderModal({
        control: {
          ...createMockHostMdmProfile({
            name: "Host name",
            platform: "darwin",
            operation_type: null,
            detail: "",
          }),
          profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
          status: "verified",
        },
      });

      expect(
        screen.getByText(/was renamed/, { selector: "span" })
      ).toHaveTextContent(
        "Anna's MacBook Pro was renamed to match this fleet's host name template. Fleet verified."
      );
    });

    it("names the host in the recovery lock password message", () => {
      renderModal({
        control: generateRecoveryLockPasswordSetting("verified", ""),
      });

      expect(
        screen.getByText(/recovery lock password/, { selector: "span" })
      ).toHaveTextContent(
        "Fleet set a recovery lock password for Anna's MacBook Pro."
      );
    });

    it("names the host and certificate in the android certificate message", () => {
      renderModal({
        control: {
          ...createMockHostMdmProfile({
            name: "Wi-Fi cert",
            platform: "android",
            operation_type: "install",
            detail: "",
          }),
          profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
          status: "verified",
        },
      });

      expect(
        screen.getByText(/applied/, { selector: "span" })
      ).toHaveTextContent(
        "Anna's MacBook Pro applied Wi-Fi cert. Fleet verified."
      );
    });

    // Windows reaches action_required for a missing PIN, an unready TPM, and a policy that forbids a TPM-only
    // protector, so the fallback must not name any one of them. It is only reached if the server omits a reason.
    it("falls back to neutral wording on windows disk encryption with no reason", () => {
      renderModal({
        control: generateWinDiskEncryptionSetting("action_required", ""),
      });

      expect(
        screen.getByText(/needs attention/, { selector: "span" })
      ).toHaveTextContent(
        "Disk encryption on Anna's MacBook Pro needs attention."
      );
      expect(screen.queryByText(/BitLocker PIN/)).toBeNull();
    });

    // The generic copy names a PIN, but the server reaches action_required for
    // reasons that have nothing to do with one. When it sends a reason, that
    // reason wins, otherwise we assert a PIN requirement that may not exist.
    it("prefers the server reason over the PIN wording on windows disk encryption", () => {
      const detail =
        "BitLocker protection is off. Fleet could not turn it back on: could not add a TPM protector, so protection was not re-enabled: 0x80310066";

      renderModal({
        control: generateWinDiskEncryptionSetting("action_required", detail),
      });

      expect(
        screen.getByText(/could not turn it back on/, { selector: "span" })
      ).toHaveTextContent(detail);
      expect(screen.queryByText(/hasn't set a BitLocker PIN/)).toBeNull();
    });
  });

  describe("non-failed controls carrying a detail", () => {
    // A custom activation's predicate can exclude a host. Fleet delivered the
    // profile, so the status is Verified, but "applied the setting" is untrue.
    it("replaces the status message on a verified control", () => {
      const detail =
        'Fleet verified, but predicate ("ASSET-002" == "ASSET-001") evaluated to false and settings were not applied to this host.';

      renderModal({
        control: createMockHostMdmProfile({
          name: "Passcode",
          platform: "darwin",
          operation_type: "install",
          status: "verified",
          detail,
        }),
      });

      expect(screen.getByText(detail)).toBeInTheDocument();
      expect(
        screen.queryByText(/applied Passcode\. Fleet verified\./)
      ).not.toBeInTheDocument();
      expect(screen.queryByText("Details:")).not.toBeInTheDocument();
    });

    // Resending only nulls the status, so until the next cron run a Windows
    // control reads Enforcing while still carrying the previous attempt's
    // output. The state has to stay legible and the output copyable.
    it("keeps the status message and shows the detail below on an enforcing control", () => {
      renderModal({
        control: createMockHostMdmProfile({
          name: "DeviceLock",
          platform: "windows",
          operation_type: "install",
          status: "pending",
          detail:
            "./Device/Vendor/MSFT/Policy/Config/DeviceLock/PreventLockScreenSlideShow: status 500, ./Device/Vendor/MSFT/Policy/Config/DeviceLock/PreventForestFires: status 200",
        }),
      });

      expect(
        screen.getByText(/is running the MDM command to apply/, {
          selector: "span",
        })
      ).toHaveTextContent(
        "Anna's MacBook Pro is running the MDM command to apply DeviceLock or will run it when the host comes online."
      );
      expect(screen.getByText("Details:")).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Copy details" })
      ).toBeInTheDocument();
    });

    it("keeps the status message and shows the detail below on an android control waiting for a certificate", () => {
      const detail =
        'Waiting for certificate "WiFi-Cert" to be installed on the host before applying this profile.';

      renderModal({
        control: createMockHostMdmProfile({
          name: "01-wifi-eap-tls.onc",
          platform: "android",
          operation_type: "install",
          status: "pending",
          detail,
        }),
      });

      expect(
        screen.getByText(/is running the MDM command to apply/, {
          selector: "span",
        })
      ).toBeInTheDocument();
      expect(screen.getByText(detail)).toBeInTheDocument();
    });

    it("does not mention osquery for ddm verifying controls", () => {
      renderModal({
        control: createMockHostMdmProfile({
          name: "DeviceLock",
          platform: "darwin",
          operation_type: "install",
          status: "verifying",
          detail: "Some detail about the verifying status",
          profile_uuid: `dbogus-uuid`,
        }),
      });

      expect(
        screen.getByText(/acknowledged the MDM command to apply/, {
          selector: "span",
        })
      ).toBeInTheDocument();
      expect(
        screen.queryByText(/verifying with osquery/)
      ).not.toBeInTheDocument();
    });
  });
});
