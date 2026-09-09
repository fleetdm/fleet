import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";
import { renderWithSetup } from "test/test-utils";

import { FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID } from "interfaces/mdm";
import {
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
  REC_LOCK_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

import OSSettingsResendCell from "./OSSettingsResendCell";

const noop = () => Promise.resolve();

describe("OSSettingsResendCell", () => {
  it("renders a resend button when canResendProfiles is true and profile is failed", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles
        canRotateRecoveryLockPassword={false}
        profile={createMockHostMdmProfile({ status: "failed" })}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Resend" })).toBeInTheDocument();
  });

  it("renders a resend button when canResendProfiles is true and profile is verified", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles
        canRotateRecoveryLockPassword={false}
        profile={createMockHostMdmProfile({ status: "verified" })}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Resend" })).toBeInTheDocument();
  });

  it("renders a rotate button when canRotateRecoveryLockPassword is true and password status is verified", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles={false}
        canRotateRecoveryLockPassword
        profile={createMockHostMdmProfile({
          profile_uuid: REC_LOCK_SYNTHETIC_PROFILE_UUID,
          status: "verified",
        })}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Rotate" })).toBeInTheDocument();
  });

  it("renders a rotate button when canRotateRecoveryLockPassword is true and password status is failed", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles={false}
        canRotateRecoveryLockPassword
        profile={createMockHostMdmProfile({
          profile_uuid: REC_LOCK_SYNTHETIC_PROFILE_UUID,
          status: "failed",
        })}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Rotate" })).toBeInTheDocument();
  });

  it("does not render a rotate button when canRotateRecoveryLockPassword is false", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles={false}
        canRotateRecoveryLockPassword={false}
        profile={createMockHostMdmProfile({
          profile_uuid: REC_LOCK_SYNTHETIC_PROFILE_UUID,
          status: "verified",
        })}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(
      screen.queryByRole("button", { name: "Rotate" })
    ).not.toBeInTheDocument();
  });

  it("does not render a rotate button when password status is pending", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles={false}
        canRotateRecoveryLockPassword
        profile={createMockHostMdmProfile({
          profile_uuid: REC_LOCK_SYNTHETIC_PROFILE_UUID,
          status: "pending",
        })}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(
      screen.queryByRole("button", { name: "Rotate" })
    ).not.toBeInTheDocument();
  });

  describe("Android certificate row", () => {
    it.each(["delivering", "delivered"] as const)(
      "renders a resend button when the certificate is stuck in %s (Enforcing)",
      (status) => {
        render(
          <OSSettingsResendCell
            canResendProfiles
            profile={createMockHostMdmProfile({
              platform: "android",
              profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
              certificate_template_id: 1,
              operation_type: "install",
              status,
            })}
            resendRequest={noop}
          />
        );

        expect(
          screen.getByRole("button", { name: "Resend" })
        ).toBeInTheDocument();
      }
    );

    it("does not render a resend button when the certificate is still pending", () => {
      render(
        <OSSettingsResendCell
          canResendProfiles
          profile={createMockHostMdmProfile({
            platform: "android",
            profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
            certificate_template_id: 1,
            operation_type: "install",
            status: "pending",
          })}
          resendRequest={noop}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resend" })
      ).not.toBeInTheDocument();
    });

    it("does not treat a non-certificate Android profile stuck in delivering as resendable", () => {
      render(
        <OSSettingsResendCell
          canResendProfiles
          profile={createMockHostMdmProfile({
            platform: "android",
            profile_uuid: "some-other-profile-uuid",
            operation_type: "install",
            status: "delivering",
          })}
          resendRequest={noop}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resend" })
      ).not.toBeInTheDocument();
    });
  });

  it("shows a disabled resend button with a tooltip for an Android configuration profile", async () => {
    const { user } = renderWithSetup(
      <OSSettingsResendCell
        canResendProfiles={false}
        showDisabledResendForAndroidProfile
        profile={createMockHostMdmProfile({
          platform: "android",
          status: "failed",
        })}
        resendRequest={noop}
      />
    );

    const resendButton = screen.getByRole("button", { name: "Resend" });
    expect(resendButton).toBeDisabled();

    await user.hover(resendButton);

    expect(
      await screen.findByText(/Fleet can't resend this configuration profile/i)
    ).toBeInTheDocument();
  });

  it("does not show a disabled resend button when showDisabledResendForAndroidProfile is false", () => {
    render(
      <OSSettingsResendCell
        canResendProfiles={false}
        showDisabledResendForAndroidProfile={false}
        profile={createMockHostMdmProfile({
          platform: "android",
          status: "failed",
        })}
        resendRequest={noop}
      />
    );

    expect(
      screen.queryByRole("button", { name: "Resend" })
    ).not.toBeInTheDocument();
  });

  describe("host name template row", () => {
    it("renders a resend button when canResendHostNameTemplate is true and status is failed", () => {
      render(
        <OSSettingsResendCell
          canResendProfiles={false}
          canResendHostNameTemplate
          profile={createMockHostMdmProfile({
            profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
            status: "failed",
          })}
          resendRequest={noop}
          resendHostNameTemplate={noop}
        />
      );

      expect(
        screen.getByRole("button", { name: "Resend" })
      ).toBeInTheDocument();
    });

    it("renders a resend button when canResendHostNameTemplate is true and status is verified", () => {
      render(
        <OSSettingsResendCell
          canResendProfiles={false}
          canResendHostNameTemplate
          profile={createMockHostMdmProfile({
            profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
            status: "verified",
          })}
          resendRequest={noop}
          resendHostNameTemplate={noop}
        />
      );

      expect(
        screen.getByRole("button", { name: "Resend" })
      ).toBeInTheDocument();
    });

    it("does not render a resend button when status is pending", () => {
      render(
        <OSSettingsResendCell
          canResendProfiles={false}
          canResendHostNameTemplate
          profile={createMockHostMdmProfile({
            profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
            status: "pending",
          })}
          resendRequest={noop}
          resendHostNameTemplate={noop}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resend" })
      ).not.toBeInTheDocument();
    });

    it("does not render a resend button when canResendHostNameTemplate is false (e.g. device user page)", () => {
      render(
        <OSSettingsResendCell
          canResendProfiles={false}
          canResendHostNameTemplate={false}
          profile={createMockHostMdmProfile({
            profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
            status: "failed",
          })}
          resendRequest={noop}
        />
      );

      expect(
        screen.queryByRole("button", { name: "Resend" })
      ).not.toBeInTheDocument();
    });

    it("calls resendHostNameTemplate (not resendRequest) when clicked", async () => {
      const resendHostNameTemplate = jest.fn(() => Promise.resolve());
      const resendRequest = jest.fn(() => Promise.resolve());

      render(
        <OSSettingsResendCell
          canResendProfiles={false}
          canResendHostNameTemplate
          profile={createMockHostMdmProfile({
            profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
            status: "failed",
          })}
          resendRequest={resendRequest}
          resendHostNameTemplate={resendHostNameTemplate}
        />
      );

      fireEvent.click(screen.getByRole("button", { name: "Resend" }));

      await waitFor(() => {
        expect(resendHostNameTemplate).toHaveBeenCalledTimes(1);
      });
      expect(resendRequest).not.toHaveBeenCalled();
    });
  });
});
