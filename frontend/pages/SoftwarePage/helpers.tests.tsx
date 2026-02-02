import React from "react";
import { render, screen } from "@testing-library/react";

import { ISoftwareInstallPolicy } from "interfaces/software";

import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreApp,
} from "__mocks__/softwareMock";
import {
  createMockHostSoftwarePackage,
  createMockHostAppStoreApp,
  createMockHostSoftware,
} from "__mocks__/hostMock";
import {
  getSelfServiceTooltip,
  getAutomaticInstallPoliciesCount,
  getDisplayedSoftwareName,
} from "./helpers";

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

    expect(
      screen.getByText(/End users can install from self-service\./i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Learn how to deploy self-service/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Learn how to deploy self-service/i })
    ).toHaveAttribute(
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
      screen.getByRole("link", { name: /Learn more/i })
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Learn more/i })).toHaveAttribute(
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

  it("returns a default when neither name nor display_name is provided", () => {
    expect(getDisplayedSoftwareName(undefined, undefined)).toBe("Software");
    expect(getDisplayedSoftwareName(null, null)).toBe("Software");
  });
});
