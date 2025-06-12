import React, { MouseEvent } from "react";

import Button from "components/buttons/Button";
import { hyphenateString } from "utilities/strings/stringUtils";

interface RenderYamlHelperText {
  installScript?: string;
  uninstallScript?: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  onClickPreInstallQuery?: (evt: MouseEvent) => void;
  onClickInstallScript?: (evt: MouseEvent) => void;
  onClickPostInstallScript?: (evt: MouseEvent) => void;
  onClickUninstallScript?: (evt: MouseEvent) => void;
}

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

export const renderDownloadFilesText = ({
  preInstallQuery,
  installScript,
  postInstallScript,
  uninstallScript,
  onClickPreInstallQuery,
  onClickInstallScript,
  onClickPostInstallScript,
  onClickUninstallScript,
}: RenderYamlHelperText): JSX.Element => {
  const items: { key: string; element: JSX.Element }[] = [];

  if (preInstallQuery) {
    items.push({
      key: "pre-install-query",
      element: (
        <Button key="pre" variant="text-link" onClick={onClickPreInstallQuery}>
          pre-install query
        </Button>
      ),
    });
  }
  if (installScript) {
    items.push({
      key: "install-script",
      element: (
        <Button
          key="install"
          variant="text-link"
          onClick={onClickInstallScript}
        >
          install script
        </Button>
      ),
    });
  }
  if (uninstallScript) {
    items.push({
      key: "uninstall-script",
      element: (
        <Button
          key="uninstall"
          variant="text-link"
          onClick={onClickUninstallScript}
        >
          uninstall script
        </Button>
      ),
    });
  }
  if (postInstallScript) {
    items.push({
      key: "post-install-script",
      element: (
        <Button
          key="post"
          variant="text-link"
          onClick={onClickPostInstallScript}
        >
          post-install script
        </Button>
      ),
    });
  }

  if (items.length === 0) return <></>;

  return (
    <>
      Next, download your {joinWithCommasAnd(items)} and add{" "}
      {items.length === 1 ? "it" : "them"} to your repository using the{" "}
      {items.length === 1 ? "path" : "paths"} above. If you edited{" "}
      <b>Advanced options</b>, download and replace the{" "}
      {items.length === 1 ? "file" : "files"} in your repository with the
      updated {items.length === 1 ? "one" : "ones"}.
    </>
  );
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
}

export const createPackageYaml = ({
  softwareTitle,
  packageName,
  version,
  url,
  sha256,
  preInstallQuery,
  installScript,
  postInstallScript,
  uninstallScript,
}: CreatePackageYamlParams): string => {
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

  const hyphenatedSWTitle = hyphenateString(softwareTitle);

  if (preInstallQuery) {
    yaml += `pre_install_query:
  path: ../queries/pre-install-query-${hyphenatedSWTitle}.yml
`;
  }

  if (installScript) {
    yaml += `install_script:
  path: ../scripts/install-${hyphenatedSWTitle}.sh
`;
  }

  if (postInstallScript) {
    yaml += `post_install_script:
  path: ../scripts/post-install-${hyphenatedSWTitle}.sh
`;
  }

  if (uninstallScript) {
    yaml += `uninstall_script:
  path: ../scripts/uninstall-${hyphenatedSWTitle}.sh
`;
  }

  return yaml.trim();
};
