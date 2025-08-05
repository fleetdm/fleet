import React from "react";

import { dateAgo } from "utilities/date_format";

import {
  formatSoftwareType,
  IHostSoftware,
  ISoftwareInstallVersion,
  SoftwareSource,
} from "interfaces/software";

import Card from "components/Card";
import DataSet from "components/DataSet";

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
  bundleIdentifier?: string;
}

const InventoryVersion = ({
  version,
  source,
  bundleIdentifier,
}: IInventoryVersionProps) => {
  const {
    vulnerabilities,
    installed_paths: installedPaths,
    signature_information: signatureInformation,
  } = version;

  return (
    <Card
      className={`${baseClass}__version`}
      color="grey"
      borderRadiusSize="medium"
    >
      <div className={`${baseClass}__row`}>
        <DataSet title="Version" value={version.version} />
        <DataSet title="Type" value={formatSoftwareType({ source })} />
        {bundleIdentifier && (
          <DataSet title="Bundle identifier" value={bundleIdentifier} />
        )}
        {version.last_opened_at ||
        source === "programs" ||
        source === "apps" ? (
          <DataSet
            title="Last opened"
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
              value={formatSoftwareType({ source: hostSoftware.source })}
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
            />
          );
        })}
      </div>
    </div>
  );
};

export default InventoryVersions;
