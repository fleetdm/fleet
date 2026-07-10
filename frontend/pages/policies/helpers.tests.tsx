import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
  createMockAppStoreApp,
} from "__mocks__/softwareMock";

import { generateSoftwareOptionHelpText } from "./helpers";

describe("generateSoftwareOptionHelpText", () => {
  it("builds the platform/extension and version subtitle for a package", () => {
    const title = createMockSoftwareTitle({
      source: "programs",
      app_store_app: null,
      software_package: createMockSoftwarePackage({
        name: "Notepad++.exe",
        version: "8.9.6.4",
      }),
    });

    expect(generateSoftwareOptionHelpText(title)).toBe(
      "Windows (.exe) • 8.9.6.4"
    );
  });

  it("maps the source to its platform display name (macOS package)", () => {
    const title = createMockSoftwareTitle({
      source: "apps",
      app_store_app: null,
      software_package: createMockSoftwarePackage({
        name: "TestPackage-1.2.3.pkg",
        version: "1.2.3",
      }),
    });

    expect(generateSoftwareOptionHelpText(title)).toBe("macOS (.pkg) • 1.2.3");
  });

  it("labels App Store (VPP) apps and uses the app_store_app version", () => {
    const title = createMockSoftwareTitle({
      source: "apps",
      app_store_app: createMockAppStoreApp({ version: "5.0.0" }),
    });

    expect(generateSoftwareOptionHelpText(title)).toBe(
      "macOS (App Store) • 5.0.0"
    );
  });

  it("omits the version for a VPP app that has none", () => {
    const title = createMockSoftwareTitle({
      source: "apps",
      // default app_store_app mock has no `version`
      app_store_app: createMockAppStoreApp(),
    });

    expect(generateSoftwareOptionHelpText(title)).toBe("macOS (App Store)");
  });

  it("omits the version when the package has none", () => {
    const title = createMockSoftwareTitle({
      source: "programs",
      app_store_app: null,
      software_package: createMockSoftwarePackage({
        name: "MyApp.exe",
        version: "",
      }),
    });

    expect(generateSoftwareOptionHelpText(title)).toBe("Windows (.exe)");
  });

  it("omits the leading separator when there is a version but no platform string", () => {
    const title = createMockSoftwareTitle({
      // null-platform source -> empty platform string, but the package has a version
      source: "go_binaries",
      app_store_app: null,
      software_package: createMockSoftwarePackage({
        name: "mytool",
        version: "1.2.3",
      }),
    });

    expect(generateSoftwareOptionHelpText(title)).toBe("1.2.3");
  });

  it("returns an empty string when neither platform nor version is available", () => {
    const title = createMockSoftwareTitle({
      source: "programs",
      app_store_app: null,
      software_package: null,
    });

    expect(generateSoftwareOptionHelpText(title)).toBe("");
  });
});
