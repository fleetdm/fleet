import React from "react";

import Button from "components/buttons/Button";

import { ISoftwarePackage } from "interfaces/software";

export const renderYamlHelperText = (
  softwarePackage: ISoftwarePackage
): JSX.Element | null => {
  const items: { key: string; element: JSX.Element }[] = [];

  if (softwarePackage.pre_install_query) {
    items.push({
      key: "pre-install-query",
      element: (
        <Button key="pre" variant="text-link">
          pre-install query
        </Button>
      ),
    });
  }
  if (softwarePackage.install_script) {
    items.push({
      key: "install-script",
      element: (
        <Button key="install" variant="text-link">
          install script
        </Button>
      ),
    });
  }
  if (softwarePackage.uninstall_script) {
    items.push({
      key: "uninstall-script",
      element: (
        <Button key="uninstall" variant="text-link">
          uninstall script
        </Button>
      ),
    });
  }
  if (softwarePackage.post_install_script) {
    items.push({
      key: "post-install-script",
      element: (
        <Button key="post" variant="text-link">
          post-install script
        </Button>
      ),
    });
  }

  if (items.length === 0) return null;

  // Helper to join items with commas and Oxford comma before "and"
  const joinWithCommasAnd = (
    elements: { key: string; element: JSX.Element }[]
  ) => {
    return elements.map((item, idx) => {
      if (idx === 0) return item.element;
      if (idx === elements.length - 1) {
        return (
          <React.Fragment key={`and-${item.key}`}>
            {elements.length > 2 ? "," : ""} and {item.element}
          </React.Fragment>
        );
      }
      return (
        <React.Fragment key={`comma-${item.key}`}>
          , {item.element}
        </React.Fragment>
      );
    });
  };

  return (
    <>
      Next, download your {joinWithCommasAnd(items)} and add{" "}
      {items.length === 1 ? "it" : "them"} to your repository (see above for{" "}
      {items.length === 1 ? "path" : "paths"}).
    </>
  );
};

interface CreatePackageYamlParams {
  softwareTitle: string;
  packageName: string;
  version: string;
  url?: string;
  sha256?: string | null;
  includePreInstallQuery?: boolean;
  includeInstallScript?: boolean;
  includePostInstallScript?: boolean;
  includeUninstallScript?: boolean;
}

export const createPackageYaml = ({
  softwareTitle,
  packageName,
  version,
  url,
  sha256,
  includePreInstallQuery,
  includeInstallScript,
  includePostInstallScript,
  includeUninstallScript,
}: CreatePackageYamlParams): string => {
  // Hyphenate the name for file paths
  const hyphenatedSoftwareTitle = softwareTitle
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "-");

  let yaml = `# ${softwareTitle} (${packageName}) version ${version}
`;

  if (url) {
    yaml += `url: ${url}
`;
  }

  if (sha256) {
    yaml += `hash_sha256: ${sha256}
`;
  }

  if (includePreInstallQuery) {
    yaml += `pre_install_query:
  path: ../queries/pre-install-query-${hyphenatedSoftwareTitle}.yml
`;
  }

  if (includeInstallScript) {
    yaml += `install_script:
  path: ../scripts/install-${hyphenatedSoftwareTitle}.sh
`;
  }

  if (includePostInstallScript) {
    yaml += `post_install_script:
  path: ../scripts/post-install-${hyphenatedSoftwareTitle}.sh
`;
  }

  if (includeUninstallScript) {
    yaml += `uninstall_script:
  path: ../scripts/uninstall-${hyphenatedSoftwareTitle}.sh
`;
  }

  return yaml.trim();
};
