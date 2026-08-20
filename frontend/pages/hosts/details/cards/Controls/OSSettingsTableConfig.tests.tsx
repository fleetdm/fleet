import { IHostMdmData, IHostMdmHostNameSetting } from "interfaces/host";
import { createMockHostMdmProfile } from "__mocks__/hostMock";
import {
  generateRecoveryLockPasswordSetting,
  generateWinDiskEncryptionSetting,
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

import {
  countFailedControls,
  generateTableData,
  getRowActionProps,
} from "./OSSettingsTableConfig";

const createMockHostMdmData = (
  overrides?: Partial<IHostMdmData>
): IHostMdmData => ({
  encryption_key_available: false,
  enrollment_status: "On (manual)",
  server_url: "https://example.com",
  profiles: [],
  device_status: "unlocked",
  pending_action: "",
  os_settings: {
    disk_encryption: { status: null, detail: "" },
    certificates: [],
  },
  ...overrides,
});

describe("generateTableData - host name row", () => {
  const hostNameSetting: IHostMdmHostNameSetting = {
    status: "pending",
    detail: "",
  };

  it.each(["darwin", "ios", "ipados"])(
    "appends the host name row for %s hosts when os_settings.host_name is present",
    (platform) => {
      const mdmData = createMockHostMdmData({
        os_settings: {
          disk_encryption: { status: null, detail: "" },
          certificates: [],
          host_name: hostNameSetting,
        },
      });

      const rows = generateTableData(mdmData, platform) ?? [];

      const hostNameRow = rows.find(
        (r) => r.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID
      );
      expect(hostNameRow).toBeDefined();
      expect(hostNameRow?.name).toBe("Host name");
      expect(hostNameRow?.status).toBe("pending");
    }
  );

  it.each(["darwin", "ios", "ipados"])(
    "does not append the host name row for %s hosts when os_settings.host_name is omitted",
    (platform) => {
      const mdmData = createMockHostMdmData();

      const rows = generateTableData(mdmData, platform) ?? [];

      expect(
        rows.find((r) => r.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID)
      ).toBeUndefined();
    }
  );

  it.each(["darwin", "ios", "ipados"])(
    "does not append the host name row for %s hosts that are not enrolled in MDM",
    (platform) => {
      const mdmData = createMockHostMdmData({
        enrollment_status: "Off",
        os_settings: {
          disk_encryption: { status: null, detail: "" },
          certificates: [],
          host_name: hostNameSetting,
        },
      });

      const rows = generateTableData(mdmData, platform) ?? [];

      expect(
        rows.find((r) => r.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID)
      ).toBeUndefined();
    }
  );

  it("does not append the host name row for non-Apple platforms even if host_name is present", () => {
    const mdmData = createMockHostMdmData({
      os_settings: {
        disk_encryption: { status: null, detail: "" },
        certificates: [],
        host_name: hostNameSetting,
      },
    });

    const windowsRows = generateTableData(mdmData, "windows") ?? [];
    const linuxRows = generateTableData(mdmData, "ubuntu") ?? [];

    expect(
      windowsRows.find(
        (r) => r.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID
      )
    ).toBeUndefined();
    expect(
      linuxRows.find((r) => r.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID)
    ).toBeUndefined();
  });

  it("keeps existing profiles alongside the host name row for ios hosts", () => {
    const mdmData = createMockHostMdmData({
      profiles: [
        {
          profile_uuid: "abc-123",
          name: "Wi-Fi",
          operation_type: "install",
          platform: "ios",
          status: "verified",
          detail: "",
          scope: "device",
          managed_local_account: null,
        },
      ],
      os_settings: {
        disk_encryption: { status: null, detail: "" },
        certificates: [],
        host_name: hostNameSetting,
      },
    });

    const rows = generateTableData(mdmData, "ios") ?? [];

    expect(rows).toHaveLength(2);
    expect(rows.map((r) => r.name)).toEqual(["Wi-Fi", "Host name"]);
  });
});

describe("countFailedControls", () => {
  it("counts nothing when there are no rows", () => {
    expect(countFailedControls(null)).toBe(0);
    expect(countFailedControls([])).toBe(0);
  });

  it("counts synthesized rows, not just profiles from the API", () => {
    const rows = generateTableData(
      createMockHostMdmData({
        profiles: [
          createMockHostMdmProfile({ profile_uuid: "a", status: "failed" }),
          createMockHostMdmProfile({ profile_uuid: "b", status: "verified" }),
        ],
        os_settings: {
          disk_encryption: { status: "failed", detail: "encryption failed" },
          certificates: [],
        },
      }),
      "windows"
    );

    // The failed profile plus the synthesized failed disk encryption row.
    expect(countFailedControls(rows)).toBe(2);
  });

  it("counts the synthesized host name row", () => {
    const rows = generateTableData(
      createMockHostMdmData({
        profiles: [],
        os_settings: {
          disk_encryption: { status: null, detail: "" },
          certificates: [],
          host_name: { status: "failed", detail: "drifted" },
        },
      }),
      "darwin"
    );

    expect(countFailedControls(rows)).toBe(1);
  });
});

describe("getRowActionProps", () => {
  // The synthesized rows carry placeholder UUIDs that the profile resend
  // endpoint rejects with a 404, so they must never offer Resend.
  it.each(["verified", "failed"] as const)(
    "does not offer resend on a %s windows disk encryption row",
    (status) => {
      const row = generateWinDiskEncryptionSetting(status, "");

      expect(getRowActionProps(row, true).canResendProfiles).toBe(false);
    }
  );

  it("does not offer resend on a recovery lock row, which rotates instead", () => {
    const row = generateRecoveryLockPasswordSetting("failed", "");

    expect(getRowActionProps(row, true, true)).toMatchObject({
      canResendProfiles: false,
      canRotateRecoveryLockPassword: true,
    });
  });

  it("offers resend on a real windows profile", () => {
    const row = createMockHostMdmProfile({
      profile_uuid: "w1234",
      platform: "windows",
      operation_type: "install",
      status: "failed",
    });

    expect(getRowActionProps(row, true).canResendProfiles).toBe(true);
  });
});
