import { render, screen } from "@testing-library/react";
import React from "react";

import { notify } from "components/ToastNotification";
import {
  createMockHostSoftwarePackage,
  createMockHostAppStoreApp,
  createMockHostSoftware,
} from "__mocks__/hostMock";
import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreApp,
  createMockAppStoreAppVersion,
} from "__mocks__/softwareMock";
import { ISoftwareInstallPolicy } from "interfaces/software";

import {
  getSelfServiceTooltip,
  getAutomaticInstallPoliciesCount,
  getAutoUpdateWindowDurationMinutes,
  getDefaultAutoUpdateFromVersions,
  getDisplayedSoftwareName,
  routeVersionNameError,
} from "./helpers";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

describe("getSelfServiceTooltip", () => {
  it("returns Play Store tooltip content when isAndroidPlayStoreApp is true", () => {
    const tooltip = getSelfServiceTooltip(false, true);

    render(tooltip as React.ReactElement);

    expect(
      screen.getByText(/End users can install from the/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/Play Store/i)).toBeInTheDocument();
    expect(screen.getByText(/in their work profile\./i)).toBeInTheDocument();
  });

  it("returns iOS self-service tooltip content when isIosOrIpadosApp is true and isAndroidPlayStoreApp is false", () => {
    const tooltip = getSelfServiceTooltip(true, false);

    render(tooltip as React.ReactElement);

    // "self service" is the link text now — the sentence wraps it inline.
    expect(screen.getByText(/End users can install from/i)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /self service/i })
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /self service/i })).toHaveAttribute(
      "href",
      expect.stringContaining("/deploy-self-service-to-ios")
    );
  });

  it("returns Fleet Desktop self-service tooltip when both flags are false", () => {
    const tooltip = getSelfServiceTooltip(false, false);

    render(tooltip as React.ReactElement);

    expect(screen.getByText(/End users can install from/i)).toBeInTheDocument();
    expect(screen.getByText(/Fleet Desktop/i)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Self service/i })
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Self service/i })).toHaveAttribute(
      "href",
      expect.stringContaining("/self-service-software")
    );
  });
});

// Helper to create an array of dummy policies
const makePolicies = (count: number): ISoftwareInstallPolicy[] =>
  Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    name: `Policy ${i + 1}`,
    type: i % 2 === 0 ? "patch" : "dynamic" /* alternate types for variety */,
  }));

describe("getAutomaticInstallPoliciesCount", () => {
  const policyCounts = [0, 1, 3];

  policyCounts.forEach((count) => {
    describe(`when there are ${count} automatic install policies`, () => {
      it(`returns ${count} for software_package (ISoftwareTitle)`, () => {
        const softwareTitle = createMockSoftwareTitle({
          software_package: {
            ...createMockSoftwarePackage(),
            automatic_install_policies: makePolicies(count),
          },
          app_store_app: null,
        });
        expect(getAutomaticInstallPoliciesCount(softwareTitle)).toBe(count);
      });

      it(`returns ${count} for app_store_app (ISoftwareTitle)`, () => {
        const softwareTitle = createMockSoftwareTitle({
          software_package: null,
          app_store_app: {
            ...createMockAppStoreApp(),
            automatic_install_policies: makePolicies(count),
          },
        });
        expect(getAutomaticInstallPoliciesCount(softwareTitle)).toBe(count);
      });

      it(`returns ${count} for software_package (IHostSoftware)`, () => {
        const hostSoftware = createMockHostSoftware({
          software_package: {
            ...createMockHostSoftwarePackage(),
            automatic_install_policies: makePolicies(count),
          },
          app_store_app: null,
        });
        expect(getAutomaticInstallPoliciesCount(hostSoftware)).toBe(count);
      });

      it(`returns ${count} for app_store_app (IHostSoftware)`, () => {
        const hostSoftware = createMockHostSoftware({
          software_package: null,
          app_store_app: {
            ...createMockHostAppStoreApp(),
            automatic_install_policies: makePolicies(count),
          },
        });
        expect(getAutomaticInstallPoliciesCount(hostSoftware)).toBe(count);
      });
    });
  });

  it("returns 0 if neither software_package nor app_store_app is present (IHostSoftware)", () => {
    const hostSoftware = createMockHostSoftware({
      software_package: null,
      app_store_app: null,
    });
    expect(getAutomaticInstallPoliciesCount(hostSoftware)).toBe(0);
  });

  it("returns 0 if neither software_package nor app_store_app is present (ISoftwareTitle)", () => {
    const hostSoftware = createMockSoftwareTitle({
      software_package: null,
      app_store_app: null,
    });
    expect(getAutomaticInstallPoliciesCount(hostSoftware)).toBe(0);
  });
});

describe("getDisplayedSoftwareName", () => {
  it("returns display_name when provided (custom name wins)", () => {
    expect(
      getDisplayedSoftwareName("Microsoft.CompanyPortal", "My Custom Portal")
    ).toBe("My Custom Portal");
  });

  it("normalizes a known raw name when display_name is not provided", () => {
    expect(getDisplayedSoftwareName("Microsoft.CompanyPortal", null)).toBe(
      "Company Portal"
    );
  });

  it("normalizes a known raw name case-insensitively", () => {
    expect(getDisplayedSoftwareName("microsoft.companyportal", undefined)).toBe(
      "Company Portal"
    );
  });

  it("returns the raw name when it is not in WELL_KNOWN_SOFTWARE_TITLES", () => {
    expect(getDisplayedSoftwareName("Some Other App", null)).toBe(
      "Some Other App"
    );
  });

  it("returns the raw name when display_name is empty", () => {
    expect(getDisplayedSoftwareName("Some App", "")).toBe("Some App");
  });

  it("treats a whitespace-only display_name as absent and falls back to name", () => {
    expect(getDisplayedSoftwareName("Some App", " ")).toBe("Some App");
  });

  it("returns the default when display_name and name are both whitespace-only", () => {
    expect(getDisplayedSoftwareName(" ", " ")).toBe("Software");
  });

  it("returns a default when neither name nor display_name is provided", () => {
    expect(getDisplayedSoftwareName(undefined, undefined)).toBe("Software");
    expect(getDisplayedSoftwareName(null, null)).toBe("Software");
  });

  // macOS hides helper apps such as MediaRemoteUI by setting CFBundleDisplayName
  // to a lone U+200E LEFT-TO-RIGHT MARK. trim() does not strip it, so the name
  // arrives here as a non-empty but entirely invisible string.
  it.each([
    ["left-to-right mark", "\u200e"],
    ["right-to-left mark", "\u200f"],
    ["zero-width space", "\u200b"],
    ["word joiner", "\u2060"],
    ["right-to-left override", "\u202e"],
    ["byte order mark", "\ufeff"],
    ["non-breaking space", "\u00a0"],
    ["null character", "\u0000"],
    ["delete character", "\u007f"],
    ["mixed invisibles", "\u200e\u00a0\u200b"],
  ])("labels a name of only %s with the bundle identifier", (_label, name) => {
    expect(
      getDisplayedSoftwareName(name, null, "com.apple.MediaRemoteUI")
    ).toBe("com.apple.MediaRemoteUI");
  });

  it("treats an invisible display_name as absent and falls back to name", () => {
    expect(getDisplayedSoftwareName("Some App", "\u200e")).toBe("Some App");
  });

  it("returns the default when the name is invisible and there is no bundle identifier", () => {
    expect(getDisplayedSoftwareName("\u200e", null)).toBe("Software");
  });

  it("returns the default when the bundle identifier is also invisible", () => {
    expect(getDisplayedSoftwareName("\u200e", null, "\u200e")).toBe("Software");
  });

  // Only wholly invisible names fall back; a name with visible characters is
  // left untouched so legitimate titles are not rewritten.
  it("keeps a name that merely contains an invisible character", () => {
    expect(getDisplayedSoftwareName("Foo\u200eBar", null)).toBe("Foo\u200eBar");
  });

  it("prefers display_name over the bundle identifier", () => {
    expect(
      getDisplayedSoftwareName("\u200e", "My App", "com.apple.MediaRemoteUI")
    ).toBe("My App");
  });
});

describe("getAutoUpdateWindowDurationMinutes", () => {
  it("returns same-day duration when end > start", () => {
    expect(getAutoUpdateWindowDurationMinutes("09:00", "10:30")).toBe(90);
  });

  it("returns zero when start equals end", () => {
    expect(getAutoUpdateWindowDurationMinutes("12:00", "12:00")).toBe(0);
  });

  it("wraps overnight when end < start", () => {
    // 23:00 to 00:00 = 60 minutes (the minimum valid overnight window).
    expect(getAutoUpdateWindowDurationMinutes("23:00", "00:00")).toBe(60);
  });

  // The pre-fix regression: 23:30 to 00:15 is 45 minutes overnight but the
  // naive `end >= start && end - start < 60` check skipped the branch, letting
  // the sub-hour window through client-side validation.
  it("catches short overnight windows (23:30 to 00:15 = 45 minutes)", () => {
    expect(getAutoUpdateWindowDurationMinutes("23:30", "00:15")).toBe(45);
  });

  it("handles a full-length overnight window", () => {
    // 22:00 to 02:00 = 240 minutes.
    expect(getAutoUpdateWindowDurationMinutes("22:00", "02:00")).toBe(240);
  });

  it("returns null for malformed inputs", () => {
    expect(getAutoUpdateWindowDurationMinutes("bad", "02:00")).toBeNull();
    expect(getAutoUpdateWindowDurationMinutes("09:00", "25:00")).toBeNull();
    expect(getAutoUpdateWindowDurationMinutes("", "")).toBeNull();
  });
});

describe("getDefaultAutoUpdateFromVersions", () => {
  it("returns undefined when there are no versions", () => {
    expect(getDefaultAutoUpdateFromVersions([])).toBeUndefined();
    expect(getDefaultAutoUpdateFromVersions(null)).toBeUndefined();
    expect(getDefaultAutoUpdateFromVersions(undefined)).toBeUndefined();
  });

  it("returns the shared schedule when all versions agree", () => {
    const a = createMockAppStoreAppVersion({
      auto_update_enabled: true,
      auto_update_window_start: "22:00",
      auto_update_window_end: "02:00",
    });
    const b = createMockAppStoreAppVersion({
      auto_update_enabled: true,
      auto_update_window_start: "22:00",
      auto_update_window_end: "02:00",
    });
    expect(getDefaultAutoUpdateFromVersions([a, b])).toEqual({
      enabled: true,
      windowStart: "22:00",
      windowEnd: "02:00",
    });
  });

  it("returns undefined when siblings disagree on the window", () => {
    const a = createMockAppStoreAppVersion({
      auto_update_enabled: true,
      auto_update_window_start: "22:00",
      auto_update_window_end: "02:00",
    });
    const b = createMockAppStoreAppVersion({
      auto_update_enabled: true,
      auto_update_window_start: "23:00",
      auto_update_window_end: "02:00",
    });
    expect(getDefaultAutoUpdateFromVersions([a, b])).toBeUndefined();
  });

  it("returns undefined when siblings disagree on enabled", () => {
    const a = createMockAppStoreAppVersion({
      auto_update_enabled: true,
      auto_update_window_start: "22:00",
      auto_update_window_end: "02:00",
    });
    const b = createMockAppStoreAppVersion({
      auto_update_enabled: false,
      auto_update_window_start: "22:00",
      auto_update_window_end: "02:00",
    });
    expect(getDefaultAutoUpdateFromVersions([a, b])).toBeUndefined();
  });

  it("normalizes nullish windows to empty strings when all versions agree", () => {
    const a = createMockAppStoreAppVersion({
      auto_update_enabled: false,
      auto_update_window_start: null,
      auto_update_window_end: null,
    });
    expect(getDefaultAutoUpdateFromVersions([a])).toEqual({
      enabled: false,
      windowStart: "",
      windowEnd: "",
    });
  });
});

describe("routeVersionNameError", () => {
  const setServerErrors = jest.fn();
  const notifyError = notify.error as jest.Mock;

  beforeEach(() => {
    setServerErrors.mockClear();
    notifyError.mockClear();
  });

  // Regression guard: the duplicate-version-name conflict must land on the
  // Name field, not the toast. Matcher keys on the BE phrase "a version named"
  // (see server/datastore/mysql/vpp.go and ee/server/service/vpp.go — reword
  // those and this test breaks).
  it("routes `A version named ...` to the Name field", () => {
    const err = {
      response: {
        data: {
          errors: [
            {
              name: "base",
              reason:
                "A version named \"Production\" already exists for this app in this fleet.",
            },
          ],
        },
      },
    };

    routeVersionNameError(err, setServerErrors, "fallback");

    expect(setServerErrors).toHaveBeenCalledWith({
      name:
        "A version named \"Production\" already exists for this app in this fleet.",
    });
    expect(notifyError).not.toHaveBeenCalled();
  });

  // Regression guard: an unrelated error whose reason happens to contain
  // "name" (e.g. an unsupported FLEET_VAR username) must stay in the toast,
  // not clobber the Name field.
  it("routes an unrelated `name`-containing reason to the toast", () => {
    const err = {
      response: {
        data: {
          errors: [
            {
              name: "base",
              reason:
                "Unsupported variable $FLEET_VAR_HOST_END_USER_EMAIL_IDP_USERNAME",
            },
          ],
        },
      },
    };

    routeVersionNameError(err, setServerErrors, "Couldn't update. Please try again.");

    expect(setServerErrors).not.toHaveBeenCalled();
    expect(notifyError).toHaveBeenCalledWith(
      "Couldn't update. Please try again.",
      { response: err }
    );
  });
});
