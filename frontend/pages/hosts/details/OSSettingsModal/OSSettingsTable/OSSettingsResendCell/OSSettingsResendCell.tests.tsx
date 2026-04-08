import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";

import { REC_LOCK_SYNTHETIC_PROFILE_UUID } from "pages/hosts/details/helpers";

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
});
