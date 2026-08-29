import { getLastFetchedTime } from "./helpers";

describe("getLastFetchedTime", () => {
  it("uses policy_updated_at when it is more recent than detail_updated_at (#patch-when-closed skip activity timestamp mismatch)", () => {
    const detailUpdatedAt = "2026-08-28T22:05:16Z";
    const policyUpdatedAt = "2026-08-28T22:11:27Z";

    expect(getLastFetchedTime(detailUpdatedAt, policyUpdatedAt)).toEqual(
      policyUpdatedAt
    );
  });

  it("uses detail_updated_at when it is more recent than policy_updated_at", () => {
    const detailUpdatedAt = "2026-08-28T22:11:27Z";
    const policyUpdatedAt = "2026-08-28T22:05:16Z";

    expect(getLastFetchedTime(detailUpdatedAt, policyUpdatedAt)).toEqual(
      detailUpdatedAt
    );
  });

  it("falls back to whichever timestamp is present when the other is missing", () => {
    expect(getLastFetchedTime(undefined, "2026-08-28T22:11:27Z")).toEqual(
      "2026-08-28T22:11:27Z"
    );
    expect(getLastFetchedTime("2026-08-28T22:11:27Z", undefined)).toEqual(
      "2026-08-28T22:11:27Z"
    );
  });

  it("returns undefined when neither timestamp is set", () => {
    expect(getLastFetchedTime(undefined, undefined)).toBeUndefined();
  });
});
