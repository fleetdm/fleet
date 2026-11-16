import { render, screen } from "@testing-library/react";
import React from "react";

import type { IRegistrationFormData } from "interfaces/registration_form_data";
import { getPastDate, getFutureDate } from "test/test-utils";

import helpers, {
  getHostOSDisplayName,
  compareVersions,
  willExpireWithinXDays,
  humanLastSeen,
  humanHostMemory,
  internationalTimeOnlyFormat,
  formatSelectedTargetsForApi,
  formatFleet5RemovalMessage,
} from "./helpers";

describe("helpers utilities", () => {
  describe("getHostOSDisplayName function", () => {
    it.each([
      ["macOS", "14.1.2", "macOS 14.1.2"],
      ["iPadOS", "17.5.1", "iPadOS 17.5.1"],
      ["Ubuntu", "24.04.5 LTS", "Ubuntu 24.04.5 LTS"],
      ["Android", "16 (2026-05-01)", "Android 16 (2026-05-01)"],
      [
        "Windows Server 2022 Datacenter 21H2 (Server Core)",
        "10.0.20348.1850",
        "Windows Server 2022 Datacenter 21H2 (Server Core) 10.0.20348.1850",
      ],
      ["Omarchy", "4.0.0", "Omarchy 4.0.0"],
      ["", "", ""],
      [undefined, undefined, ""],
      ["---", "---", ""],
    ])("joins %p and %p into %p", (osName, osVersion, expected) => {
      expect(getHostOSDisplayName(osName, osVersion)).toEqual(expected);
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

  describe("humanHostMemory function", () => {
    it("returns --- when the memory is unavailable", () => {
      expect(humanHostMemory(-1)).toEqual("---");
      expect(humanHostMemory(-1024)).toEqual("---");
    });

    it("formats bytes in GB", () => {
      expect(humanHostMemory(8589934592)).toEqual("8.0 GB");
      expect(humanHostMemory(0)).toEqual("0.0 GB");
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

  describe("formatSelectedTargetsForApi function", () => {
    it("sends fleet targets under `fleets`", () => {
      const targets = [
        { id: 2, target_type: "hosts" },
        { id: 3, target_type: "labels" },
        { id: 9, target_type: "teams" },
        { id: 4, target_type: "teams" },
      ] as Parameters<typeof formatSelectedTargetsForApi>[0];

      expect(formatSelectedTargetsForApi(targets)).toEqual({
        hosts: [2],
        labels: [3],
        fleets: [4, 9],
      });
    });
  });

  describe("internationalTimeOnlyFormat function", () => {
    const setLanguage = (lang: string) => {
      Object.defineProperty(window.navigator, "languages", {
        value: [lang],
        configurable: true,
      });
    };

    let originalLanguages: readonly string[];
    beforeAll(() => {
      originalLanguages = window.navigator.languages;
    });
    afterAll(() => {
      Object.defineProperty(window.navigator, "languages", {
        value: originalLanguages,
        configurable: true,
      });
    });

    it("renders 24-hour source times in 12-hour form for US locale", () => {
      setLanguage("en-US");
      // A narrow no-break space (U+202F) or NBSP (U+00A0) may appear
      // between the number and AM/PM in some ICU versions; match any
      // whitespace via \\s? to stay portable across Node/ICU versions.
      expect(internationalTimeOnlyFormat("02:00")).toMatch(/2:00\s?AM/i);
      expect(internationalTimeOnlyFormat("14:30")).toMatch(/2:30\s?PM/i);
    });

    it("keeps 24-hour form for a 24-hour locale (de-DE)", () => {
      setLanguage("de-DE");
      // German locale uses 24-hour; either 02:00 or 2:00 is acceptable
      // depending on ICU version, but always without AM/PM.
      expect(internationalTimeOnlyFormat("02:00")).toMatch(/^0?2:00$/);
      expect(internationalTimeOnlyFormat("14:30")).toMatch(/^14:30$/);
    });

    it("passes through an unparseable string unchanged", () => {
      setLanguage("en-US");
      expect(internationalTimeOnlyFormat("nope")).toBe("nope");
      expect(internationalTimeOnlyFormat("")).toBe("");
      expect(internationalTimeOnlyFormat("2:")).toBe("2:");
    });

    it("passes through out-of-range times unchanged", () => {
      setLanguage("en-US");
      // These would otherwise silently wrap via Date.setHours (24:00 -> next day).
      // Passthrough keeps the tooltip honest about bad data instead of hiding it.
      expect(internationalTimeOnlyFormat("24:00")).toBe("24:00");
      expect(internationalTimeOnlyFormat("10:60")).toBe("10:60");
    });

    it("handles edge times 00:00 and 23:59", () => {
      setLanguage("en-US");
      expect(internationalTimeOnlyFormat("00:00")).toMatch(/12:00\s?AM/i);
      expect(internationalTimeOnlyFormat("23:59")).toMatch(/11:59\s?PM/i);
    });
  });

  describe("formatFleet5RemovalMessage function", () => {
    const guideURL = "https://fleetdm.com/learn-more-about/fleet-5-migration";

    it("returns null for any other message", () => {
      expect(formatFleet5RemovalMessage("Couldn't add. Learn more")).toBeNull();
    });

    it("links the guide and keeps the text around it", () => {
      render(
        <div>
          {formatFleet5RemovalMessage(
            `The "$FLEET_VAR_HOST_END_USER_EMAIL_IDP" variable is not available anymore. What to use instead: ${guideURL} Trailing note.`
          )}
        </div>
      );

      expect(screen.getByRole("link", { name: guideURL })).toHaveAttribute(
        "href",
        guideURL
      );
      expect(
        screen.getByText(/variable is not available anymore/)
      ).toBeInTheDocument();
      expect(screen.getByText(/Trailing note\./)).toBeInTheDocument();
    });
  });
});
