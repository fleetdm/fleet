import { render, screen } from "@testing-library/react";
import { ISoftwarePackage } from "interfaces/software";
import { createMockSoftwarePackage } from "__mocks__/softwareMock";
import { createPackageYaml, renderYamlHelperText } from "./helpers";

describe("createPackageYaml", () => {
  it("generates YAML with all fields present", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Google Chrome",
      packageName: "googlechrome.pkg",
      version: "136.0.1",
      url: "https://dl.google.com/chrome.pkg",
      sha256: "abc123",
      includePreInstallQuery: true,
      includeInstallScript: true,
      includePostInstallScript: true,
      includeUninstallScript: true,
    });

    expect(yaml).toBe(`# Google Chrome (googlechrome.pkg) version 136.0.1
url: https://dl.google.com/chrome.pkg
hash_sha256: abc123
pre_install_query:
  path: ../queries/pre-install-query-google-chrome.yml
install_script:
  path: ../scripts/install-google-chrome.sh
post_install_script:
  path: ../scripts/post-install-google-chrome.sh
uninstall_script:
  path: ../scripts/uninstall-google-chrome.sh`);
  });

  it("omits optional fields when not provided", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Slack",
      packageName: "slack.pkg",
      version: "5.0.0",
      // url and sha256 not provided
      includePreInstallQuery: false,
      includeInstallScript: false,
      includePostInstallScript: false,
      includeUninstallScript: false,
    });

    expect(yaml).toBe(`# Slack (slack.pkg) version 5.0.0`);
  });

  it("handles some scripts/queries provided", () => {
    const yaml = createPackageYaml({
      softwareTitle: "Firefox",
      packageName: "firefox.pkg",
      version: "120.0",
      includePreInstallQuery: true,
      includeInstallScript: false,
      includePostInstallScript: true,
      includeUninstallScript: false,
    });

    expect(yaml).toBe(`# Firefox (firefox.pkg) version 120.0
pre_install_query:
  path: ../queries/pre-install-query-firefox.yml
post_install_script:
  path: ../scripts/post-install-firefox.sh`);
  });

  it("hyphenates name correctly for file paths", () => {
    const yaml = createPackageYaml({
      softwareTitle: "My Cool App",
      packageName: "mycoolapp.pkg",
      version: "1.2.3",
      includeInstallScript: true,
    });

    expect(yaml).toBe(`# My Cool App (mycoolapp.pkg) version 1.2.3
install_script:
  path: ../scripts/install-my-cool-app.sh`);
  });

  it("does not include hash_sha256 if sha256 is null or empty", () => {
    const yamlNull = createPackageYaml({
      softwareTitle: "Null Hash",
      packageName: "nullhash.pkg",
      version: "0.0.1",
      sha256: null,
      includeInstallScript: true,
    });

    const yamlEmpty = createPackageYaml({
      softwareTitle: "Empty Hash",
      packageName: "emptyhash.pkg",
      version: "0.0.2",
      sha256: "",
      includeInstallScript: true,
    });

    expect(yamlNull).toBe(`# Null Hash (nullhash.pkg) version 0.0.1
install_script:
  path: ../scripts/install-null-hash.sh`);
    expect(yamlEmpty).toBe(`# Empty Hash (emptyhash.pkg) version 0.0.2
install_script:
  path: ../scripts/install-empty-hash.sh`);
  });
});

describe("renderYamlHelperText", () => {
  it("renders nothing if no scripts/queries are present", () => {
    // Explicitly override all to undefined or empty to simulate 'no items'
    const pkg: ISoftwarePackage = createMockSoftwarePackage({
      install_script: undefined,
      uninstall_script: undefined,
      pre_install_query: undefined,
      post_install_script: undefined,
    });
    const { container } = render(renderYamlHelperText(pkg));
    expect(container).toBeEmptyDOMElement();
  });

  it("renders correctly with one item", () => {
    // Only install_script present
    const pkg: ISoftwarePackage = createMockSoftwarePackage({
      uninstall_script: undefined,
      pre_install_query: undefined,
      post_install_script: undefined,
    });
    render(renderYamlHelperText(pkg));
    expect(
      screen.getByRole("button", { name: "install script" })
    ).toBeInTheDocument();
    expect(
      screen.getByText((content) =>
        content.includes("add it to your repository (see above for path).")
      )
    ).toBeInTheDocument();
    expect(screen.queryByText("and")).not.toBeInTheDocument();
  });

  it("renders correctly with two items", () => {
    // install_script and uninstall_script present
    const pkg: ISoftwarePackage = createMockSoftwarePackage({
      pre_install_query: undefined,
      post_install_script: undefined,
    });

    const { container } = render(renderYamlHelperText(pkg));
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
        content.includes("add them to your repository (see above for paths).")
      )
    ).toBeInTheDocument();
  });

  it("renders correctly with all items", () => {
    // All present (default)
    const pkg: ISoftwarePackage = createMockSoftwarePackage();

    const { container } = render(renderYamlHelperText(pkg));
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
        content.includes("add them to your repository (see above for paths).")
      )
    ).toBeInTheDocument();
  });

  it("renders comma correctly for three items (with Oxford comma)", () => {
    // pre_install_query, install_script, uninstall_script present
    const pkg: ISoftwarePackage = createMockSoftwarePackage({
      post_install_script: undefined,
    });

    const { container } = render(renderYamlHelperText(pkg));
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
        content.includes("add them to your repository (see above for paths).")
      )
    ).toBeInTheDocument();
  });
});
