import { getPastDate, getFutureDate } from "test/test-utils";
import type { IRegistrationFormData } from "interfaces/registration_form_data";
import helpers, {
  removeOSPrefix,
  compareVersions,
  willExpireWithinXDays,
  humanLastSeen,
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

    it("compares segments numerically instead of lexically, regardless of digit count", () => {
      expect(compareVersions("26.6", "26.10")).toEqual(-1);
      expect(compareVersions("26.10", "26.6")).toEqual(1);
      expect(compareVersions("150.0.7871.213", "150.0.7871.185")).toEqual(1);
      expect(compareVersions("10.0.26200.8875", "10.0.9200.100")).toEqual(1);
    });

    it("treats a version with a non-numeric first segment as older than any comparable version", () => {
      expect(compareVersions("rolling", "26.6")).toEqual(-1);
      expect(compareVersions("26.6", "rolling")).toEqual(1);
      expect(compareVersions("rolling", "rolling")).toEqual(0);
      // Narrow case that a whole-segment-count-matching NaN tie used to get wrong:
      // a single bare numeric segment vs. a single non-numeric one.
      expect(compareVersions("rolling", "14")).toEqual(-1);
      expect(compareVersions("14", "rolling")).toEqual(1);
    });

    it("compares Windows feature-update codenames by year and half", () => {
      expect(compareVersions("21H2", "22H1")).toEqual(-1);
      expect(compareVersions("22H1", "21H2")).toEqual(1);
      expect(compareVersions("22H1", "22H2")).toEqual(-1);
      expect(compareVersions("23H1", "21H2")).toEqual(1);
      expect(compareVersions("21H2", "21H2")).toEqual(0);
    });

    it("leaves suffixed Fleet-maintained-app versions with a numeric first segment unchanged", () => {
      // A non-numeric trailing segment (e.g. a Homebrew revision suffix) is
      // coerced to 0 by the `|| 0` fallback below, same as a missing
      // segment — this is pre-existing behavior this change doesn't alter,
      // since only the *first* segment's numeric-ness is checked up front.
      expect(compareVersions("2.26.7_1", "2.26.7")).toEqual(-1);
      expect(compareVersions("114.0.4-release", "114.0.4")).toEqual(-1);
      // A differing leading numeric segment still compares correctly even
      // with a non-numeric suffix present.
      expect(compareVersions("2.27.0_1", "2.26.7_1")).toEqual(1);
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

  describe("humanLastSeen function", () => {
    beforeEach(() => {
      jest.useFakeTimers().setSystemTime(new Date("2026-06-15T12:00:00Z"));
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it("uses days below the month threshold", () => {
      expect(humanLastSeen(getPastDate(5))).toEqual("5 days ago");
      expect(humanLastSeen(getPastDate(89))).toEqual("89 days ago");
    });

    it("uses months at or beyond 90 days", () => {
      expect(humanLastSeen(getPastDate(90))).toEqual("3 months ago");
      expect(humanLastSeen(getPastDate(100))).toEqual("3 months ago");
    });
  });

  describe("setupData function", () => {
    it("excludes the org logo file from the JSON setup payload", () => {
      const formData: IRegistrationFormData = {
        email: "admin@example.com",
        name: "Admin",
        password: "password123",
        password_confirmation: "password123",
        org_name: "Fleet",
        org_web_url: "",
        org_logo_file: new File(["x"], "logo.png", { type: "image/png" }),
        fleet_web_address: "",
        server_url: "https://fleet.example.com",
      };

      const result = helpers.setupData(formData);

      expect(result.org_info).toEqual({ org_name: "Fleet" });
    });
  });
});
