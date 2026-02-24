import { createMockSoftwarePackage } from "__mocks__/softwareMock";

import { createPackageYaml } from "./helpers";

describe("createPackageYaml", () => {
  const {
    name,
    version,
    url,
    icon_url: iconUrl,
    display_name: displayName,
    hash_sha256: sha256,
    pre_install_query: preInstallQuery,
    install_script: installScript,
    post_install_script: postInstallScript,
    uninstall_script: uninstallScript,
  } = createMockSoftwarePackage();

  it("generates YAML with all fields present", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Falcon Sensor Test Package",
      packageName: name,
      iconUrl,
      displayName,
      version,
      url,
      sha256,
      preInstallQuery,
      installScript,
      postInstallScript,
      uninstallScript,
    });

    expect(yaml)
      .toBe(`# Falcon Sensor Test Package (TestPackage-1.2.3.pkg) version 1.2.3
- url: https://fakeurl.testpackageurlforfalconapp.fake/test/package
  hash_sha256: abcd1234
  pre_install_query:
    path: ../queries/pre-install-query-falcon-sensor-test-package.yml
  install_script:
    path: ../scripts/install-falcon-sensor-test-package.sh
  post_install_script:
    path: ../scripts/post-install-falcon-sensor-test-package.sh
  uninstall_script:
    path: ../scripts/uninstall-falcon-sensor-test-package.sh`);
  });

  it("omits optional fields when not provided", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Falcon Sensor Test Package",
      packageName: name,
      iconUrl,
      displayName,
      version,
      url: undefined,
      sha256: undefined,
      preInstallQuery: undefined,
      installScript: undefined,
      postInstallScript: undefined,
      uninstallScript: undefined,
    });

    expect(yaml).toBe(
      "# Falcon Sensor Test Package (TestPackage-1.2.3.pkg) version 1.2.3"
    );
  });

  it("handles some scripts/queries provided", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Falcon Sensor Test Package",
      packageName: name,
      iconUrl,
      displayName,
      version,
      url: undefined,
      sha256: undefined,
      preInstallQuery,
      installScript: undefined,
      postInstallScript,
      uninstallScript: undefined,
    });

    expect(yaml)
      .toBe(`# Falcon Sensor Test Package (TestPackage-1.2.3.pkg) version 1.2.3
  pre_install_query:
    path: ../queries/pre-install-query-falcon-sensor-test-package.yml
  post_install_script:
    path: ../scripts/post-install-falcon-sensor-test-package.sh`);
  });

  it("hyphenates name correctly for file paths", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Falcon Sensor Test Package",
      packageName: name,
      iconUrl,
      displayName,
      version,
      url: undefined,
      sha256: undefined,
      preInstallQuery: undefined,
      installScript,
      postInstallScript: undefined,
      uninstallScript: undefined,
    });

    expect(yaml)
      .toBe(`# Falcon Sensor Test Package (TestPackage-1.2.3.pkg) version 1.2.3
  install_script:
    path: ../scripts/install-falcon-sensor-test-package.sh`);
  });

  it("does not include hash_sha256 if sha256 is null or empty", () => {
    const yamlNull = createPackageYaml({
      softwareTitle: "Null Hash",
      packageName: name,
      iconUrl,
      displayName,
      version,
      url: undefined,
      sha256: null,
      preInstallQuery: undefined,
      installScript,
      postInstallScript: undefined,
      uninstallScript: undefined,
    });

    const yamlEmpty = createPackageYaml({
      softwareTitle: "Empty Hash",
      packageName: name,
      iconUrl,
      displayName,
      version,
      url: undefined,
      sha256: "",
      preInstallQuery: undefined,
      installScript,
      postInstallScript: undefined,
      uninstallScript: undefined,
    });

    expect(yamlNull).toBe(`# Null Hash (TestPackage-1.2.3.pkg) version 1.2.3
  install_script:
    path: ../scripts/install-null-hash.sh`);
    expect(yamlEmpty).toBe(`# Empty Hash (TestPackage-1.2.3.pkg) version 1.2.3
  install_script:
    path: ../scripts/install-empty-hash.sh`);
  });

  it("omits script-only fields for script packages", () => {
    // Script packages (.sh and .ps1) should not expose install_script,
    // post_install_script, uninstall_script, or pre_install_query
    const yaml = createPackageYaml({
      softwareTitle: "My Script Package",
      packageName: "my-script.sh",
      version: "1.0.0",
      url: "https://example.com/my-script.sh",
      sha256: "abc123",
      preInstallQuery,
      installScript,
      postInstallScript,
      uninstallScript,
      iconUrl: null,
      displayName,
      isScriptPackage: true,
    });

    // Should only include comment, url, and hash_sha256
    expect(yaml).toBe(`# My Script Package (my-script.sh) version 1.0.0
- url: https://example.com/my-script.sh
  hash_sha256: abc123`);

    // Verify it doesn't contain any of the forbidden fields
    expect(yaml).not.toContain("install_script");
    expect(yaml).not.toContain("post_install_script");
    expect(yaml).not.toContain("uninstall_script");
    expect(yaml).not.toContain("pre_install_query");
  });

  it("generates icon url and display name", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Falcon Sensor Test Package",
      packageName: name,
      iconUrl: "falcon",
      displayName: "Falcon",
      version,
      url: undefined,
      sha256,
      preInstallQuery: undefined,
      installScript: undefined,
      postInstallScript: undefined,
      uninstallScript: undefined,
    });

    expect(yaml)
      .toBe(`# Falcon Sensor Test Package (TestPackage-1.2.3.pkg) version 1.2.3
- hash_sha256: abcd1234
  display_name: Falcon
  icon:
    path: ./icons/falcon-sensor-test-package-icon.png`);
  });
});
