import {
  getExpectedCheckInIntervalMs,
  getRefetchGiveUpDelayMs,
  getRefetchGiveUpReason,
} from "./host_refetch_helpers";

describe("host_refetch_helpers - getExpectedCheckInIntervalMs", () => {
  it("uses the shorter of the two intervals plus the 60s flapping buffer", () => {
    expect(
      getExpectedCheckInIntervalMs({
        seen_time: "",
        distributed_interval: 10,
        config_tls_refresh: 3600,
      })
    ).toBe((10 + 60) * 1000);
  });
});

describe("host_refetch_helpers - getRefetchGiveUpDelayMs", () => {
  it("falls back to fallbackMs for a host with a short check-in cadence", () => {
    const host = {
      seen_time: "",
      distributed_interval: 10,
      config_tls_refresh: 10,
    };
    // (10 + 60) * 1000 * 2 = 140000, which is below this fallback.
    expect(getRefetchGiveUpDelayMs(host, 180000)).toBe(180000);
  });

  it("scales past fallbackMs for a host with a long check-in cadence", () => {
    const host = {
      seen_time: "",
      distributed_interval: 3600,
      config_tls_refresh: 3600,
    };
    expect(getRefetchGiveUpDelayMs(host, 60000)).toBe((3600 + 60) * 1000 * 2);
  });
});

describe("host_refetch_helpers - getRefetchGiveUpReason", () => {
  it("returns checkin_stalled when the host hasn't been seen since before the refetch started", () => {
    const refetchStartTime = Date.parse("2026-01-01T00:05:00Z");
    const host = { seen_time: "2026-01-01T00:00:00Z" };

    expect(getRefetchGiveUpReason(host, refetchStartTime)).toBe(
      "checkin_stalled"
    );
  });

  it("returns refetch_stalled when the host checked in after the refetch started", () => {
    const refetchStartTime = Date.parse("2026-01-01T00:00:00Z");
    const host = { seen_time: "2026-01-01T00:05:00Z" };

    expect(getRefetchGiveUpReason(host, refetchStartTime)).toBe(
      "refetch_stalled"
    );
  });
});
