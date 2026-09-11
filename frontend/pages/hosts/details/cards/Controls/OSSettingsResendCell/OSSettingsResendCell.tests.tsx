import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";

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
