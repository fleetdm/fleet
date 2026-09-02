import { IDiskEncryptionSummaryResponse } from "services/entities/disk_encryption";

import { generateTableData } from "./DiskEncryptionTableConfig";

const SUMMARY: IDiskEncryptionSummaryResponse = {
  verified: { macos: 1, windows: 2, linux: 3 },
  verifying: { macos: 4, windows: 5, linux: 6 },
  action_required: { macos: 7, windows: 8, linux: 9 },
  enforcing: { macos: 10, windows: 11, linux: 12 },
  failed: { macos: 13, windows: 14, linux: 15 },
  removing_enforcement: { macos: 16, windows: 17, linux: 18 },
};

const rowFor = (
  rows: ReturnType<typeof generateTableData>,
  status: keyof IDiskEncryptionSummaryResponse
) => rows.find((row) => row.status.value === status);

describe("generateTableData", () => {
  it("returns no rows without summary data", () => {
    expect(generateTableData("macos", undefined, 1)).toEqual([]);
  });

  it("uses the selected platform's host counts", () => {
    const rows = generateTableData("windows", SUMMARY, 1);

    expect(rows.map((row) => row.hosts)).toEqual([2, 5, 14, 8, 11, 17]);
    expect(rows.every((row) => row.teamId === 1)).toBe(true);
  });

  it("mentions the escrowed key in the default verifying and verified tooltips", () => {
    const rows = generateTableData("macos", SUMMARY, 1);

    expect(rowFor(rows, "verified")?.status.tooltip).toMatch(
      /sent their key to Fleet/
    );
    expect(rowFor(rows, "verifying")?.status.tooltip).toMatch(
      /retrieving the disk encryption key/
    );
  });

  it("drops the key phrasing when macOS is enforce-only", () => {
    const rows = generateTableData("macos", SUMMARY, 1, true);

    expect(rowFor(rows, "verified")?.status.tooltip).toBe(
      "These hosts turned disk encryption on. Fleet verified with osquery."
    );
    expect(rowFor(rows, "verifying")?.status.tooltip).toBe(
      "These hosts acknowledged the MDM command to turn on disk encryption. Fleet is verifying with osquery. This may take up to one hour."
    );
  });

  it("leaves the other status tooltips unchanged when macOS is enforce-only", () => {
    const defaults = generateTableData("macos", SUMMARY, 1);
    const enforceOnly = generateTableData("macos", SUMMARY, 1, true);

    ([
      "action_required",
      "enforcing",
      "failed",
      "removing_enforcement",
    ] as const).forEach((status) => {
      expect(rowFor(enforceOnly, status)?.status).toEqual(
        rowFor(defaults, status)?.status
      );
    });
  });
});
