import { render, screen } from "@testing-library/react";
import { createMockSoftwarePackage } from "__mocks__/softwareMock";
import { noop } from "lodash";

import {
  hyphenatedSoftwareTitle,
  createPackageYaml,
  renderYamlHelperText,
} from "./helpers";

describe("hyphenatedSoftwareTitle", () => {
  it("converts spaces to hyphens and lowercases", () => {
    expect(hyphenatedSoftwareTitle("My Cool App")).toBe("my-cool-app");
  });

  it("trims leading and trailing spaces", () => {
    expect(hyphenatedSoftwareTitle("   Leading and trailing   ")).toBe(
      "leading-and-trailing"
    );
  });

  it("collapses multiple spaces into one hyphen", () => {
    expect(hyphenatedSoftwareTitle("Multiple    spaces here")).toBe(
      "multiple-spaces-here"
    );
  });

  it("returns empty string for empty input", () => {
    expect(hyphenatedSoftwareTitle("")).toBe("");
  });

  it("handles already hyphenated and lowercase input", () => {
    expect(hyphenatedSoftwareTitle("already-hyphenated-title")).toBe(
      "already-hyphenated-title"
    );
  });

  it("handles single word", () => {
    expect(hyphenatedSoftwareTitle("Word")).toBe("word");
  });

  it("handles all uppercase", () => {
    expect(hyphenatedSoftwareTitle("ALL UPPERCASE")).toBe("all-uppercase");
  });

  it("handles mixed case and spaces", () => {
    expect(hyphenatedSoftwareTitle("  MixED CaSe   App ")).toBe(
      "mixed-case-app"
    );
  });
});

describe("createPackageYaml", () => {
  const {
    name,
    version,
    url,
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
url: https://fakeurl.testpackageurlforfalconapp.fake/test/package
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
});

describe("renderYamlHelperText", () => {
  const {
    pre_install_query: preInstallQuery,
    install_script: installScript,
    post_install_script: postInstallScript,
    uninstall_script: uninstallScript,
  } = createMockSoftwarePackage();

  it("renders nothing if no scripts/queries are present", () => {
    // Empty to simulate 'no items'
    const { container } = render(renderYamlHelperText({}));
    expect(container).toBeEmptyDOMElement();
  });

  it("renders correctly with one item", () => {
    // Only install_script present
    render(renderYamlHelperText({ installScript, onClickInstallScript: noop }));
    expect(
      screen.getByRole("button", { name: "install script" })
    ).toBeInTheDocument();
    expect(
      screen.getByText((content) =>
        content.includes(
          "add it to your repository (please use the above path)."
        )
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("and")).not.toBeInTheDocument();
  });

  it("renders correctly with two items", () => {
    const { container } = render(
      renderYamlHelperText({
        installScript,
        uninstallScript,
        onClickInstallScript: noop,
        onClickUninstallScript: noop,
      })
    );
    expect(
      screen.getByRole("button", { name: "install script" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "uninstall script" })
    ).toBeInTheDocument();

    // In "Next," only
    const text = container.textContent ?? "";
    const commaCount = (text.match(/,/g) || []).length;
    expect(commaCount).toBe(1);

    // No oxford comma for two items
    expect(
      screen.queryByText((content) => content.includes(", and"))
    ).not.toBeInTheDocument();
    expect(
      screen.getByText((content) =>
        content.includes(
          "add them to your repository (please use the above paths)."
        )
      )
    ).toBeInTheDocument();
  });

  it("renders correctly with all items", () => {
    // All present (default)
    const { container } = render(
      renderYamlHelperText({
        preInstallQuery,
        installScript,
        uninstallScript,
        postInstallScript,
        onClickPreInstallQuery: noop,
        onClickInstallScript: noop,
        onClickUninstallScript: noop,
        onClickPostInstallScript: noop,
      })
    );
    expect(
      screen.getByRole("button", { name: "pre-install query" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "install script" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "post-install script" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "uninstall script" })
    ).toBeInTheDocument();

    // In "Next," and 3 more commas
    const text = container.textContent ?? "";
    const commaCount = (text.match(/,/g) || []).length;
    expect(commaCount).toBe(4);

    // Oxford comma for four items
    expect(
      screen.queryByText((content) => content.includes(", and"))
    ).toBeInTheDocument();
    expect(
      screen.getByText((content) =>
        content.includes(
          "add them to your repository (please use the above paths)."
        )
      )
    ).toBeInTheDocument();
  });

  it("renders comma correctly for three items (with Oxford comma)", () => {
    // pre_install_query, install_script, uninstall_script present
    const { container } = render(
      renderYamlHelperText({
        preInstallQuery,
        installScript,
        uninstallScript,
        onClickPreInstallQuery: noop,
        onClickInstallScript: noop,
        onClickUninstallScript: noop,
      })
    );

    expect(
      screen.getByRole("button", { name: "pre-install query" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "install script" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "uninstall script" })
    ).toBeInTheDocument();

    // In "Next," and 2 more commas
    const text = container.textContent ?? "";
    const commaCount = (text.match(/,/g) || []).length;
    expect(commaCount).toBe(3);

    // Oxford comma for three items
    expect(
      screen.getByText((content) => content.includes(", and"))
    ).toBeInTheDocument();
    expect(
      screen.getByText((content) =>
        content.includes(
          "add them to your repository (please use the above paths)."
        )
      )
    ).toBeInTheDocument();
  });
});
