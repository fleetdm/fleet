import { hyphenateString } from "utilities/strings/stringUtils";

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

  // Script packages don't emit install_script (the file is the install script);
  // they do support pre_install_query, post_install_script, and uninstall_script.
  if (preInstallQuery) {
    yaml += `  pre_install_query:
    path: ../queries/pre-install-query-${hyphenatedSWTitle}.yml
`;
  }

  if (!isScriptPackage && installScript) {
    yaml += `  install_script:
    path: ../scripts/install-${hyphenatedSWTitle}.sh
`;
  }

  if (postInstallScript) {
    yaml += `  post_install_script:
    path: ../scripts/post-install-${hyphenatedSWTitle}.sh
`;
  }

  if (uninstallScript) {
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
