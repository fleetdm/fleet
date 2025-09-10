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
import { getAutomaticInstallPoliciesCount } from "./helpers";

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
