import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
import { FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID } from "interfaces/mdm";
import {
  generateRecoveryLockPasswordSetting,
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

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
