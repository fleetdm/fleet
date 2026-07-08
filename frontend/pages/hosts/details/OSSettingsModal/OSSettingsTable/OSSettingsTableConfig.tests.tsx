import { IHostMdmData } from "interfaces/host";
import { HOST_NAME_SYNTHETIC_PROFILE_UUID } from "pages/hosts/details/helpers";

import { generateTableData } from "./OSSettingsTableConfig";

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
  const hostNameSetting = {
    status: "pending" as const,
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
