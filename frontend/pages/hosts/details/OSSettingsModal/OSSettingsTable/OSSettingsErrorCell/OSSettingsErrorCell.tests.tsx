import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostMdmProfile } from "__mocks__/hostMock";

import { REC_LOCK_SYNTHETIC_PROFILE_UUID } from "pages/hosts/details/helpers";

import OSSettingsErrorCell from "./OSSettingsErrorCell";

const noop = () => new Promise<void>(() => undefined);

describe("OSSettingsErrorCell", () => {
  it("renders nothing when there is no action to show", () => {
    const { container } = render(
      <OSSettingsErrorCell
        canResendProfiles={false}
        canRotateRecoveryLockPassword={false}
        profile={createMockHostMdmProfile({})}
        resendRequest={noop}
        rotateRecoveryLockPassword={noop}
      />
    );

    expect(container.innerHTML).toBe("");
  });

  it("renders a resend button when canResendProfiles is true and profile is failed", () => {
    render(
      <OSSettingsErrorCell
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
      <OSSettingsErrorCell
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
      <OSSettingsErrorCell
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
      <OSSettingsErrorCell
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
    const { container } = render(
      <OSSettingsErrorCell
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

    expect(container.innerHTML).toBe("");
  });

  it("does not render a rotate button when password status is pending", () => {
    const { container } = render(
      <OSSettingsErrorCell
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

    expect(container.innerHTML).toBe("");
  });
});
