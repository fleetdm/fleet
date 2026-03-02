import React, { MouseEvent } from "react";

import { hyphenateString } from "utilities/strings/stringUtils";

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

interface CreatePackageYamlParams {
  softwareTitle: string;
  packageName: string;
  version: string;
  url?: string;
  sha256?: string | null;
  preInstallQuery?: string;
  installScript?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  iconUrl: string | null;
  displayName?: string;
  isScriptPackage?: boolean;
}

const createPackageYaml = ({
  softwareTitle,
  packageName,
  version,
  url,
  sha256,
  preInstallQuery,
  installScript,
  postInstallScript,
  uninstallScript,
  iconUrl,
  displayName,
  isScriptPackage = false,
}: CreatePackageYamlParams): string => {
  let yaml = `# ${softwareTitle} (${packageName}) version ${version}
`;

  if (url) {
    yaml += `- url: ${url}
`;
  }

  if (sha256) {
    yaml += url ? "  " : "- ";
    yaml += `hash_sha256: ${sha256}
`;
  }

  if (displayName) {
    yaml += `  display_name: ${displayName}
`;
  }

  const hyphenatedSWTitle = hyphenateString(softwareTitle);

  // Script packages (.sh and .ps1) should not expose install_script,
  // post_install_script, uninstall_script, or pre_install_query fields.
  // The file contents themselves become the install script.
  if (!isScriptPackage && preInstallQuery) {
    yaml += `  pre_install_query:
    path: ../queries/pre-install-query-${hyphenatedSWTitle}.yml
`;
  }

  if (!isScriptPackage && installScript) {
    yaml += `  install_script:
    path: ../scripts/install-${hyphenatedSWTitle}.sh
`;
  }

  if (!isScriptPackage && postInstallScript) {
    yaml += `  post_install_script:
    path: ../scripts/post-install-${hyphenatedSWTitle}.sh
`;
  }

  if (!isScriptPackage && uninstallScript) {
    yaml += `  uninstall_script:
    path: ../scripts/uninstall-${hyphenatedSWTitle}.sh
`;
  }

  if (iconUrl) {
    yaml += `  icon:
    path: ./icons/${hyphenatedSWTitle}-icon.png
`;
  }

  return yaml.trim();
};

export default createPackageYaml;
