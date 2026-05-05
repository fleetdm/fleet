import {
  isHistoricalDataEnabled,
  IHistoricalDataSettings,
  HISTORICAL_DATA_CONFIG_KEYS,
  DATASET_CONFIG_KEY,
  DATASET_LABEL,
} from "./charts";

describe("isHistoricalDataEnabled", () => {
  const enabled: IHistoricalDataSettings = {
    uptime: true,
    vulnerabilities: true,
  };
  const uptimeOff: IHistoricalDataSettings = {
    uptime: false,
    vulnerabilities: true,
  };
  const vulnsOff: IHistoricalDataSettings = {
    uptime: true,
    vulnerabilities: false,
  };

  it("returns true when both global and fleet are undefined", () => {
    expect(isHistoricalDataEnabled(undefined, undefined, "uptime")).toBe(true);
    expect(
      isHistoricalDataEnabled(undefined, undefined, "vulnerabilities")
    ).toBe(true);
  });

  it("returns true when global is enabled and fleet is undefined", () => {
    expect(isHistoricalDataEnabled(enabled, undefined, "uptime")).toBe(true);
  });

  it("returns true when fleet is enabled and global is undefined", () => {
    expect(isHistoricalDataEnabled(undefined, enabled, "uptime")).toBe(true);
  });

  it("returns true when both global and fleet are explicitly enabled", () => {
    expect(isHistoricalDataEnabled(enabled, enabled, "uptime")).toBe(true);
    expect(isHistoricalDataEnabled(enabled, enabled, "vulnerabilities")).toBe(
      true
    );
  });

  it("returns false when only the global side is disabled", () => {
    expect(isHistoricalDataEnabled(uptimeOff, enabled, "uptime")).toBe(false);
  });

  it("returns false when only the fleet side is disabled", () => {
    expect(isHistoricalDataEnabled(enabled, uptimeOff, "uptime")).toBe(false);
  });

  it("returns false when both sides are disabled", () => {
    expect(isHistoricalDataEnabled(uptimeOff, uptimeOff, "uptime")).toBe(false);
  });

  it("evaluates per-key independently", () => {
    expect(isHistoricalDataEnabled(vulnsOff, enabled, "uptime")).toBe(true);
    expect(isHistoricalDataEnabled(vulnsOff, enabled, "vulnerabilities")).toBe(
      false
    );
  });
});

describe("HISTORICAL_DATA_CONFIG_KEYS", () => {
  it("includes uptime and vulnerabilities", () => {
    expect(HISTORICAL_DATA_CONFIG_KEYS).toEqual(["uptime", "vulnerabilities"]);
  });
});

describe("DATASET_CONFIG_KEY", () => {
  it("maps the uptime internal name to the uptime config key", () => {
    expect(DATASET_CONFIG_KEY.uptime).toBe("uptime");
  });

  it("maps the cve internal name to the vulnerabilities config key", () => {
    expect(DATASET_CONFIG_KEY.cve).toBe("vulnerabilities");
  });

  it("returns undefined for unknown internal names", () => {
    expect(DATASET_CONFIG_KEY.unknown).toBeUndefined();
  });
});

describe("DATASET_LABEL", () => {
  it("provides a human-readable label for each config key", () => {
    expect(DATASET_LABEL.uptime).toBe("Hosts online");
    expect(DATASET_LABEL.vulnerabilities).toBe("Vulnerability exposure");
  });
});
