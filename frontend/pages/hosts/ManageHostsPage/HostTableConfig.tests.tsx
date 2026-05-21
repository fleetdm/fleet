import {
  generateAvailableTableHeaders,
  generateVisibleTableColumns,
} from "./HostTableConfig";

const getIds = (columns: { id?: string }[]) => columns.map((c) => c.id ?? "");

describe("generateAvailableTableHeaders", () => {
  describe("free tier", () => {
    it("hides selection, team_name, and mdm columns when the user cannot select hosts", () => {
      const ids = getIds(
        generateAvailableTableHeaders({
          isFreeTier: true,
          canSelectHosts: false,
        })
      );

      expect(ids).not.toContain("selection");
      expect(ids).not.toContain("team_name");
      expect(ids).not.toContain("mdm.server_url");
      expect(ids).not.toContain("mdm.enrollment_status");
    });

    it("shows selection but still hides team_name and mdm columns when the user can select hosts", () => {
      const ids = getIds(
        generateAvailableTableHeaders({
          isFreeTier: true,
          canSelectHosts: true,
        })
      );

      expect(ids).toContain("selection");
      expect(ids).not.toContain("team_name");
      expect(ids).not.toContain("mdm.server_url");
      expect(ids).not.toContain("mdm.enrollment_status");
    });
  });

  describe("premium tier", () => {
    it("hides only the selection column when the user cannot select hosts", () => {
      const ids = getIds(
        generateAvailableTableHeaders({
          isFreeTier: false,
          canSelectHosts: false,
        })
      );

      expect(ids).not.toContain("selection");
      expect(ids).toContain("team_name");
      expect(ids).toContain("mdm.server_url");
      expect(ids).toContain("mdm.enrollment_status");
    });

    it("shows selection, team_name, and mdm columns when the user can select hosts", () => {
      const ids = getIds(
        generateAvailableTableHeaders({
          isFreeTier: false,
          canSelectHosts: true,
        })
      );

      expect(ids).toContain("selection");
      expect(ids).toContain("team_name");
      expect(ids).toContain("mdm.server_url");
      expect(ids).toContain("mdm.enrollment_status");
    });
  });

  it("places selection as the leading column when shown", () => {
    const ids = getIds(
      generateAvailableTableHeaders({
        isFreeTier: false,
        canSelectHosts: true,
      })
    );

    expect(ids[0]).toBe("selection");
  });
});

describe("generateVisibleTableColumns", () => {
  it("removes columns listed in hiddenColumns", () => {
    const ids = getIds(
      generateVisibleTableColumns({
        hiddenColumns: ["hostname", "computer_name", "uuid"],
        isFreeTier: false,
        canSelectHosts: true,
      })
    );

    expect(ids).not.toContain("hostname");
    expect(ids).not.toContain("computer_name");
    expect(ids).not.toContain("uuid");
    // Unrelated columns are unaffected.
    expect(ids).toContain("display_name");
    expect(ids).toContain("selection");
  });

  it("still hides the selection column when canSelectHosts is false, regardless of hiddenColumns", () => {
    const ids = getIds(
      generateVisibleTableColumns({
        hiddenColumns: [],
        isFreeTier: false,
        canSelectHosts: false,
      })
    );

    expect(ids).not.toContain("selection");
  });

  it("can hide the selection column via hiddenColumns even when canSelectHosts is true", () => {
    const ids = getIds(
      generateVisibleTableColumns({
        hiddenColumns: ["selection"],
        isFreeTier: false,
        canSelectHosts: true,
      })
    );

    expect(ids).not.toContain("selection");
  });
});
