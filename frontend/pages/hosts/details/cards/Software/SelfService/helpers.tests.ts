import {
  IDeviceSoftwareWithUiStatus,
  SoftwareCategory,
} from "interfaces/software";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";
import { createMockHostSoftwarePackage } from "__mocks__/hostMock";
import { createMockSelfServiceCategory } from "test/handlers/self-service-categories-handlers";

import {
  countUninstalledForInstallAll,
  hasInProgressInstallAllItems,
  filterCategoriesWithSoftware,
  filterSoftwareByCustomCategory,
} from "./helpers";

const makeItem = (
  ui_status: IDeviceSoftwareWithUiStatus["ui_status"],
  overrides: Partial<IDeviceSoftwareWithUiStatus> = {}
): IDeviceSoftwareWithUiStatus => ({
  ...createMockDeviceSoftware(),
  ui_status,
  ...overrides,
});

describe("countUninstalledForInstallAll", () => {
  it("counts items whose ui_status is `uninstalled`", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("uninstalled"),
        makeItem("uninstalled"),
      ])
    ).toBe(2);
  });

  it("counts `failed_install` variants as eligible (retry semantics)", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("failed_install"),
        makeItem("failed_install_installed"),
        makeItem("failed_install_update_available"),
      ])
    ).toBe(3);
  });

  it("counts `never_ran_script` as eligible (first-time script run)", () => {
    expect(countUninstalledForInstallAll([makeItem("never_ran_script")])).toBe(
      1
    );
  });

  it("excludes `installed`", () => {
    expect(countUninstalledForInstallAll([makeItem("installed")])).toBe(0);
  });

  it("excludes `recently_installed` and `recently_updated`", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("recently_installed"),
        makeItem("recently_updated"),
      ])
    ).toBe(0);
  });

  it("excludes `update_available` (user clicks Update, not Install)", () => {
    expect(countUninstalledForInstallAll([makeItem("update_available")])).toBe(
      0
    );
  });

  it("excludes `recently_uninstalled` (transient guard against immediate re-install)", () => {
    expect(
      countUninstalledForInstallAll([makeItem("recently_uninstalled")])
    ).toBe(0);
  });

  it("excludes all `failed_uninstall` variants (item still installed)", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("failed_uninstall"),
        makeItem("failed_uninstall_installed"),
        makeItem("failed_uninstall_update_available"),
      ])
    ).toBe(0);
  });

  it("excludes `ran_script` but includes `never_ran_script`", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("ran_script"),
        makeItem("never_ran_script"),
      ])
    ).toBe(1);
  });

  it("excludes in-progress statuses (installing, uninstalling, updating, running_script)", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("installing"),
        makeItem("uninstalling"),
        makeItem("updating"),
        makeItem("running_script"),
      ])
    ).toBe(0);
  });

  it("excludes pending statuses (pending_install, pending_uninstall, pending_update, pending_script)", () => {
    expect(
      countUninstalledForInstallAll([
        makeItem("pending_install"),
        makeItem("pending_uninstall"),
        makeItem("pending_update"),
        makeItem("pending_script"),
      ])
    ).toBe(0);
  });

  it("returns 0 for an empty list", () => {
    expect(countUninstalledForInstallAll([])).toBe(0);
  });
});

describe("hasInProgressInstallAllItems", () => {
  it("returns true when any item is installing", () => {
    expect(
      hasInProgressInstallAllItems([
        makeItem("uninstalled"),
        makeItem("installing"),
      ])
    ).toBe(true);
  });

  it("returns true for install/script in-flight statuses", () => {
    expect(hasInProgressInstallAllItems([makeItem("installing")])).toBe(true);
    expect(hasInProgressInstallAllItems([makeItem("running_script")])).toBe(
      true
    );
    expect(hasInProgressInstallAllItems([makeItem("pending_install")])).toBe(
      true
    );
    expect(hasInProgressInstallAllItems([makeItem("pending_script")])).toBe(
      true
    );
  });

  it("returns false for update/uninstall in-flight statuses (not install_all operations)", () => {
    expect(hasInProgressInstallAllItems([makeItem("updating")])).toBe(false);
    expect(hasInProgressInstallAllItems([makeItem("uninstalling")])).toBe(
      false
    );
    expect(hasInProgressInstallAllItems([makeItem("pending_update")])).toBe(
      false
    );
    expect(hasInProgressInstallAllItems([makeItem("pending_uninstall")])).toBe(
      false
    );
  });

  it("returns false when all items are in stable states", () => {
    expect(
      hasInProgressInstallAllItems([
        makeItem("installed"),
        makeItem("uninstalled"),
        makeItem("failed_install"),
        makeItem("recently_uninstalled"),
      ])
    ).toBe(false);
  });

  it("returns false for an empty list", () => {
    expect(hasInProgressInstallAllItems([])).toBe(false);
  });
});

describe("filterSoftwareByCustomCategory", () => {
  const browsersPackage = createMockHostSoftwarePackage({
    categories: (["🌎 Browsers"] as string[]) as SoftwareCategory[],
  });
  const securityPackage = createMockHostSoftwarePackage({
    categories: (["🔐 Security"] as string[]) as SoftwareCategory[],
  });

  const browser = makeItem("uninstalled", {
    name: "browser",
    software_package: browsersPackage,
  });
  const security = makeItem("uninstalled", {
    name: "security",
    software_package: securityPackage,
  });

  it("returns the unfiltered list when categoryId is undefined (`All` filter)", () => {
    expect(
      filterSoftwareByCustomCategory([browser, security], [], undefined)
    ).toEqual([browser, security]);
  });

  it("returns [] when categoryId is set but the categories list is empty (loading or stale URL)", () => {
    expect(filterSoftwareByCustomCategory([browser, security], [], 1)).toEqual(
      []
    );
  });

  it("returns [] when categoryId doesn't match any loaded category (stale URL)", () => {
    const categories = [
      createMockSelfServiceCategory({ id: 1, name: "🌎 Browsers" }),
    ];
    expect(
      filterSoftwareByCustomCategory([browser, security], categories, 99)
    ).toEqual([]);
  });

  it("filters items matching the selected category by name", () => {
    const categories = [
      createMockSelfServiceCategory({ id: 1, name: "🌎 Browsers" }),
    ];
    expect(
      filterSoftwareByCustomCategory([browser, security], categories, 1)
    ).toEqual([browser]);
  });

  it("matches case-insensitively", () => {
    const utilitiesPackage = createMockHostSoftwarePackage({
      categories: (["🛠️ Utilities"] as string[]) as SoftwareCategory[],
    });
    const item = makeItem("uninstalled", {
      name: "ohai",
      software_package: utilitiesPackage,
    });
    const categories = [
      createMockSelfServiceCategory({ id: 1, name: "🛠️ utilities" }),
    ];
    expect(filterSoftwareByCustomCategory([item], categories, 1)).toEqual([
      item,
    ]);
  });

  it("considers categories on app_store_app as well as software_package", () => {
    const item = makeItem("uninstalled", {
      name: "vpp-app",
      software_package: null,
      app_store_app: {
        ...createMockHostSoftwarePackage(),
        categories: ["🌎 Browsers"],
      } as never,
    });
    const categories = [
      createMockSelfServiceCategory({ id: 1, name: "🌎 Browsers" }),
    ];
    expect(filterSoftwareByCustomCategory([item], categories, 1)).toEqual([
      item,
    ]);
  });

  it("returns [] when no items match the selected category", () => {
    const categories = [
      createMockSelfServiceCategory({ id: 1, name: "🌎 Browsers" }),
    ];
    expect(filterSoftwareByCustomCategory([security], categories, 1)).toEqual(
      []
    );
  });
});

describe("filterCategoriesWithSoftware", () => {
  const browsersPackage = createMockHostSoftwarePackage({
    categories: (["🌎 Browsers"] as string[]) as SoftwareCategory[],
  });
  const securityPackage = createMockHostSoftwarePackage({
    categories: (["🔐 Security"] as string[]) as SoftwareCategory[],
  });
  const browser = makeItem("uninstalled", {
    name: "browser",
    software_package: browsersPackage,
  });
  const security = makeItem("uninstalled", {
    name: "security",
    software_package: securityPackage,
  });

  const browsers = createMockSelfServiceCategory({
    id: 1,
    name: "🌎 Browsers",
  });
  const securityCat = createMockSelfServiceCategory({
    id: 2,
    name: "🔐 Security",
  });
  const devTools = createMockSelfServiceCategory({
    id: 3,
    name: "🧰 Developer tools",
  });

  it("keeps only categories that have at least one software item", () => {
    expect(
      filterCategoriesWithSoftware(
        [browsers, securityCat, devTools],
        [browser, security]
      )
    ).toEqual([browsers, securityCat]);
  });

  it("drops every category when there is no software", () => {
    expect(
      filterCategoriesWithSoftware([browsers, securityCat, devTools], [])
    ).toEqual([]);
  });

  it("returns [] when there are no categories", () => {
    expect(filterCategoriesWithSoftware([], [browser, security])).toEqual([]);
  });

  it("matches case-insensitively", () => {
    const lowerBrowsers = createMockSelfServiceCategory({
      id: 1,
      name: "🌎 browsers",
    });
    expect(filterCategoriesWithSoftware([lowerBrowsers], [browser])).toEqual([
      lowerBrowsers,
    ]);
  });

  it("considers categories on app_store_app as well as software_package", () => {
    const vppApp = makeItem("uninstalled", {
      name: "vpp-app",
      software_package: null,
      app_store_app: {
        ...createMockHostSoftwarePackage(),
        categories: ["🌎 Browsers"],
      } as never,
    });
    expect(filterCategoriesWithSoftware([browsers], [vppApp])).toEqual([
      browsers,
    ]);
  });
});
