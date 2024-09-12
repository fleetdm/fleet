import { getPastDate, getFutureDate } from "test/test-utils";
import {
  removeOSPrefix,
  compareVersions,
  willExpireWithinXDays,
} from "./helpers";

describe("helpers utilities", () => {
  describe("removeOSPrefix function", () => {
    it("properly removes Apple prefix from a host's operating system version", () => {
      expect(removeOSPrefix("macOS 14.1.2")).toEqual("14.1.2");
      expect(removeOSPrefix("iOS 18.0")).toEqual("18.0");
      expect(removeOSPrefix("iPadOS 17.5.1")).toEqual("17.5.1");
    });
  });

  describe("compareVersions function", () => {
    it("properly checks if a version is older than another", () => {
      expect(compareVersions("14.4.1", "14.4.2")).toEqual(-1);
      expect(compareVersions("14.4.1", "14.5")).toEqual(-1);
      expect(compareVersions("14.4.1", "15")).toEqual(-1);

      expect(compareVersions("14.4", "14.4.2")).toEqual(-1);
      expect(compareVersions("14.4", "14.5")).toEqual(-1);
      expect(compareVersions("14.4", "15")).toEqual(-1);

      expect(compareVersions("14", "14.4.2")).toEqual(-1);
      expect(compareVersions("14", "14.0.5")).toEqual(-1);
      expect(compareVersions("14", "15")).toEqual(-1);
    });

    it("properly checks if a version is newer than another", () => {
      expect(compareVersions("14.4.4", "14.4.3")).toEqual(1);
      expect(compareVersions("14.3.4", "14.3")).toEqual(1);
      expect(compareVersions("14.0.4", "14")).toEqual(1);

      expect(compareVersions("14.5", "14.4.3")).toEqual(1);
      expect(compareVersions("14.5", "14.3")).toEqual(1);
      expect(compareVersions("14.5", "14")).toEqual(1);

      expect(compareVersions("14", "13.9.21")).toEqual(1);
      expect(compareVersions("14", "13.9")).toEqual(1);
      expect(compareVersions("14", "13")).toEqual(1);
    });

    it("properly checks if a version is equal to another", () => {
      expect(compareVersions("14.0.4", "14.0.4")).toEqual(0);
      expect(compareVersions("14.3", "14.3")).toEqual(0);
      expect(compareVersions("14", "14")).toEqual(0);
      expect(compareVersions("14.3", "14.3.0")).toEqual(0);
      expect(compareVersions("14", "14.0.0")).toEqual(0);
    });
  });

  describe("willExpireWithinXDays function", () => {
    it("will return true if the date is within x number of days", () => {
      const fiveDaysFromNow = getFutureDate(5);
      expect(willExpireWithinXDays(fiveDaysFromNow, 10)).toEqual(true);

      const tenDaysFromNow = getFutureDate(10);
      expect(willExpireWithinXDays(tenDaysFromNow, 30)).toEqual(true);
    });

    it("will return false if the date is not within x number of days", () => {
      const thirtyDaysFromNow = getFutureDate(30);
      expect(willExpireWithinXDays(thirtyDaysFromNow, 10)).toEqual(false);

      const fiftyDaysFromNow = getFutureDate(50);
      expect(willExpireWithinXDays(fiftyDaysFromNow, 30)).toEqual(false);
    });

    it("will return false if the date has already expired", () => {
      const fiveDaysAgo = getPastDate(5);
      expect(willExpireWithinXDays(fiveDaysAgo, 10)).toEqual(false);

      const fiftyDaysAgo = getPastDate(50);
      expect(willExpireWithinXDays(fiftyDaysAgo, 30)).toEqual(false);
    });
  });
});
