import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  IHostMdmProfile,
} from "interfaces/mdm";
import {
  generateRecoveryLockPasswordSetting,
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

import { ANDROID_CERT_RETRYING_DISPLAY_CONFIG } from "../statusDisplayConfig";
import OSSettingStatusCell from "./OSSettingStatusCell";

describe("OS setting status cell", () => {
  it("Correctly displays the status text of a profile", () => {
    render(
      <OSSettingStatusCell
        profile={createMockHostMdmProfile({
          name: "Test Profile",
          status: "verifying",
          operation_type: "install",
        })}
      />
    );

    expect(screen.getByText("Verifying")).toBeInTheDocument();
  });

  it("displays 'Removing enforcement' for a profile being removed", () => {
    render(
      <OSSettingStatusCell
        profile={createMockHostMdmProfile({
          name: "Test Profile",
          status: "pending",
          operation_type: "remove",
        })}
      />
    );

    expect(screen.getByText("Removing enforcement")).toBeInTheDocument();
  });

  it("renders 'Unrecognized' for a status with no display option", () => {
    render(
      <OSSettingStatusCell
        profile={createMockHostMdmProfile({
          name: "Test Profile",
          status: "delivered",
          operation_type: "remove",
        })}
      />
    );

    expect(screen.getByText("Unrecognized")).toBeInTheDocument();
  });

  describe("android certificate template row", () => {
    const androidCert = (
      status: "pending" | "delivering" | "delivered" | "verified" | "failed",
      operationType: "install" | "remove"
    ) =>
      createMockHostMdmProfile({
        profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
        name: "Test cert",
        platform: "android",
        status,
        operation_type: operationType,
      });

    it.each(["pending", "delivering", "delivered"] as const)(
      "displays 'Enforcing' for '%s' status with optype 'install'",
      (status) => {
        render(
          <OSSettingStatusCell profile={androidCert(status, "install")} />
        );

        expect(screen.getByText("Enforcing")).toBeInTheDocument();
      }
    );

    it.each(["delivering", "delivered"] as const)(
      "displays 'Removing enforcement' for '%s' status with optype 'remove'",
      (status) => {
        render(<OSSettingStatusCell profile={androidCert(status, "remove")} />);

        expect(screen.getByText("Removing enforcement")).toBeInTheDocument();
      }
    );
  });

  // Android certificate templates Fleet is automatically retrying after a failure. These keep an
  // in-progress status while carrying the detail from the failed attempt.
  describe("retrying android certificate template row", () => {
    const retryingCert = (
      overrides?: Partial<IHostMdmProfile>
    ): IHostMdmProfile =>
      createMockHostMdmProfile({
        profile_uuid: FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
        name: "BeyondCorp",
        platform: "android",
        operation_type: "install",
        status: "delivered",
        detail:
          "Network error during SCEP enrollment: Failed to communicate with SCEP server",
        certificate_template_id: 4,
        retrying: true,
        retry_count: 1,
        max_retries: 3,
        ...overrides,
      });

    it.each(["pending", "delivering", "delivered"] as const)(
      "displays 'Retrying' for '%s' status",
      (status) => {
        render(<OSSettingStatusCell profile={retryingCert({ status })} />);

        expect(screen.getByText("Retrying")).toBeInTheDocument();
      }
    );

    // Product decided against a new status icon for retries, since it would have to be
    // introduced across the rest of the controls UI. Retries reuse the in-progress icon and are
    // distinguished by the status text and details message alone.
    it("reuses the in-progress icon rather than introducing a new one", () => {
      expect(ANDROID_CERT_RETRYING_DISPLAY_CONFIG?.iconName).toEqual(
        "pending-outline"
      );

      const { container: retrying } = render(
        <OSSettingStatusCell profile={retryingCert()} />
      );
      const { container: enforcing } = render(
        <OSSettingStatusCell
          profile={retryingCert({
            retrying: false,
            detail: "",
            retry_count: 0,
          })}
        />
      );

      const retryingIcon = retrying.querySelector("svg");
      const enforcingIcon = enforcing.querySelector("svg");
      // Both must actually render an icon, otherwise the comparison below is vacuous.
      expect(retryingIcon).not.toBeNull();
      expect(enforcingIcon).not.toBeNull();
      expect(retryingIcon?.outerHTML).toEqual(enforcingIcon?.outerHTML);
    });

    it("displays 'Enforcing' on a first delivery that has not failed", () => {
      render(
        <OSSettingStatusCell
          profile={retryingCert({
            retrying: false,
            detail: "",
            retry_count: 0,
          })}
        />
      );

      expect(screen.getByText("Enforcing")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });

    it("displays 'Enforcing' after a manual resend, which clears the detail", () => {
      // A resend sets the retry count to the maximum so the next failure is terminal. The server
      // reports retrying: false for it, which is the only thing separating it from the final
      // automatic retry below.
      render(
        <OSSettingStatusCell
          profile={retryingCert({
            retrying: false,
            detail: "",
            retry_count: 3,
          })}
        />
      );

      expect(screen.getByText("Enforcing")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });

    // The final automatic retry looks exactly like a manual resend from the client's side: both
    // sit at the maximum retry count with no detail. Only the server can separate them, which is
    // why `retrying` is server-decided rather than inferred here.
    it("displays 'Retrying' on a final retry with no detail, unlike a resend", () => {
      render(
        <OSSettingStatusCell
          profile={retryingCert({ detail: "", retry_count: 3 })}
        />
      );

      expect(screen.getByText("Retrying")).toBeInTheDocument();
    });

    it("displays 'Removing enforcement' for a removal, which is never retried", () => {
      // The server omits the retry fields entirely on a removal rather than sending them as
      // false/zero, so the row arrives without any of them.
      render(
        <OSSettingStatusCell
          profile={retryingCert({
            retrying: undefined,
            retry_count: undefined,
            max_retries: undefined,
            operation_type: "remove",
          })}
        />
      );

      expect(screen.getByText("Removing enforcement")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });

    it("displays 'Failed' once the retries are exhausted", () => {
      render(
        <OSSettingStatusCell
          profile={retryingCert({
            retrying: false,
            status: "failed",
            retry_count: 3,
          })}
        />
      );

      expect(screen.getByText("Failed")).toBeInTheDocument();
      expect(screen.queryByText("Retrying")).not.toBeInTheDocument();
    });
  });

  describe("host name template row", () => {
    const hostNameRow = (
      status: "pending" | "verifying" | "verified" | "failed"
    ) =>
      createMockHostMdmProfile({
        profile_uuid: HOST_NAME_SYNTHETIC_PROFILE_UUID,
        name: "Host name",
        platform: "darwin",
        operation_type: null,
        status,
      });

    it.each([
      ["pending", "Enforcing"],
      ["verifying", "Verifying"],
      ["verified", "Verified"],
      ["failed", "Failed"],
    ] as const)("displays '%s' status as '%s'", (status, expected) => {
      render(<OSSettingStatusCell profile={hostNameRow(status)} />);

      expect(screen.getByText(expected)).toBeInTheDocument();
    });
  });

  describe("recovery lock password row", () => {
    it.each([
      ["pending", "Enforcing"],
      ["verified", "Verified"],
      ["failed", "Failed"],
    ] as const)("displays '%s' status as '%s'", (status, expected) => {
      render(
        <OSSettingStatusCell
          profile={generateRecoveryLockPasswordSetting(status, "")}
        />
      );

      expect(screen.getByText(expected)).toBeInTheDocument();
    });
  });

  it("displays the windows disk encryption status for the synthesized row", () => {
    render(
      <OSSettingStatusCell
        profile={createMockHostMdmProfile({
          profile_uuid: "0",
          name: "Disk encryption",
          platform: "windows",
          operation_type: null,
          status: "action_required",
        })}
      />
    );

    expect(screen.getByText("Action required")).toBeInTheDocument();
  });
});
