import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockHostMdmProfile } from "__mocks__/hostMock";

import { generateWinDiskEncryptionSetting } from "pages/hosts/details/helpers";

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

    it("renders actionable guidance above the raw detail when the error is recognized", () => {
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
      expect(screen.getByText("Details:")).toBeInTheDocument();
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

  // Introduced with custom activations and Android certificate waits: the
  // backend detail is more specific than the generic status copy.
  describe("non-failed controls carrying a detail", () => {
    const detail =
      'Fleet verified, but predicate ("ASSET-002" == "ASSET-001") evaluated to false and settings were not applied to this host.';

    it("shows the detail in place of the generic status message", () => {
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
      // The copyable block is reserved for failures.
      expect(screen.queryByText("Details:")).not.toBeInTheDocument();
    });
  });
});
