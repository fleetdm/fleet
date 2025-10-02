import React from "react";

import { dateAgo } from "utilities/date_format";

import {
  SoftwareExtensionFor,
  formatSoftwareType,
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  IHostSoftware,
  ISoftwareInstallVersion,
  SoftwareSource,
} from "interfaces/software";

import Card from "components/Card";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";

export const sourcesWithLastOpenedTime = new Set([
  "programs",
  "apps",
  "deb_packages",
  "rpm_packages",
]);

const generateVulnerabilitiesValue = (vulnerabilities: string[]) => {
  const first3 = vulnerabilities.slice(0, 3);
  const rest = vulnerabilities.slice(3);

  const first3Text = first3.join(", ");
  const restText = `, +${rest.length} more`;

  return (
    <>
      <span>{`${first3Text}${rest.length > 0 ? restText : ""}`}</span>
    </>
  );
};

const baseClass = "inventory-versions";

interface IInventoryVersionProps {
  version: ISoftwareInstallVersion;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  bundleIdentifier?: string;
}

const InventoryVersion = ({
  version,
  source,
  bundleIdentifier,
  extension_for,
}: IInventoryVersionProps) => {
  const {
    vulnerabilities,
    installed_paths: installedPaths,
    signature_information: signatureInformation,
  } = version;

  const lastOpenedTitle =
    INSTALLABLE_SOURCE_PLATFORM_CONVERSION[source] === "linux" ? (
      <TooltipWrapper
        tipContent={
          <>
            The last time the package was opened by the end user <br />
            or accessed by any process on the host.
          </>
        }
      >
        Last opened
      </TooltipWrapper>
    ) : (
      "Last opened"
    );

  return (
    <Card
      className={`${baseClass}__version`}
      color="grey"
      borderRadiusSize="medium"
    >
      <div className={`${baseClass}__row`}>
        <DataSet title="Version" value={version.version} />
        <DataSet title="Type" value={formatSoftwareType({ source, extension_for })} />
        {bundleIdentifier && (
          <DataSet title="Bundle identifier" value={bundleIdentifier} />
        )}
        {version.last_opened_at || sourcesWithLastOpenedTime.has(source) ? (
          <DataSet
            title={lastOpenedTitle}
            value={
              version.last_opened_at ? dateAgo(version.last_opened_at) : "Never"
            }
          />
        ) : null}
      </div>
      {vulnerabilities && vulnerabilities.length !== 0 && (
        <div className={`${baseClass}__row`}>
          <DataSet
            title="Vulnerabilities"
            value={generateVulnerabilitiesValue(vulnerabilities)}
          />
        </div>
      )}
      {!!installedPaths?.length &&
        installedPaths.map((path) => {
          // Find the signature info for this path
          const sigInfo = signatureInformation?.find(
            (info) => info.installed_path === path
          );

          return (
            <div className={`${baseClass}__sig-info`}>
              <DataSet orientation="horizontal" title="Path" value={path} />
              {sigInfo?.hash_sha256 && (
                <DataSet
                  orientation="horizontal"
                  title="Hash"
                  value={sigInfo.hash_sha256}
                />
              )}
            </div>
          );
        })}
    </Card>
  );
};

interface IInventoryVersionsProps {
  hostSoftware: IHostSoftware;
  showLabel?: boolean;
}
const InventoryVersions = ({
  hostSoftware,
  showLabel = true,
}: IInventoryVersionsProps) => {
  const installedVersions = hostSoftware.installed_versions;

  if (!installedVersions || installedVersions.length === 0) {
    return (
      <div className={baseClass}>
        <Card
          className={`${baseClass}__version-details`}
          color="grey"
          borderRadiusSize="medium"
        >
          <div className={`${baseClass}__row`}>
            <DataSet
              title="Type"
              value={formatSoftwareType({ source: hostSoftware.source, extension_for: hostSoftware.extension_for })}
            />
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className={baseClass}>
      {showLabel && (
        <div className={`${baseClass}__label`}>
          Current version{installedVersions.length > 1 && "s"}:
        </div>
      )}
      <div className={`${baseClass}__versions`}>
        {installedVersions.map((installedVersion) => {
          return (
            <InventoryVersion
              key={installedVersion.version}
              version={installedVersion}
              source={hostSoftware.source}
              bundleIdentifier={hostSoftware.bundle_identifier}
              extension_for={hostSoftware.extension_for}
            />
          );
        })}
      </div>
    </div>
  );
};

export default InventoryVersions;
